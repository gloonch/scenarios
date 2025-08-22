package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// Reading raw sensor data (Celsius)
type Reading struct {
	SensorID string
	Sequence int
	Celsius  float64
	At       time.Time
}

// Processed output of transform stage (Celsius + Fahrenheit)
type Processed struct {
	Reading
	Fahrenheit float64
}

// ---- Stage 0: Sensor (source) ----
// n: number of samples (if n<=0, infinite until canceled)
func Sensor(ctx context.Context, sensorID string, n int, interval time.Duration) <-chan Reading {
	out := make(chan Reading) // natural backpressure with unbuffered channel
	go func() {
		defer close(out)
		seq := 0
		t := time.NewTicker(interval)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("[sensor] canceled")
				return

			case tm := <-t.C:
				seq++
				// simulate ambient temperature around 25°C with noise
				val := 25 + rand.NormFloat64()*8
				r := Reading{SensorID: sensorID, Sequence: seq, Celsius: val, At: tm}
				log.Printf("[sensor] created reading at: %s", r.At)

				select {
				case out <- r:
				case <-ctx.Done():
					return
				}

				if n > 0 && seq >= n {
					log.Printf("[sensor] produced %d readings, closing...", n)
					return
				}
			}
		}
	}()
	return out
}

// ---- Stage 1: Filter ----
// rule: discard out-of-range values and pass only those ≥ threshold
func Filter(ctx context.Context, in <-chan Reading, min, max, threshold float64, dropped *int64) <-chan Reading {
	out := make(chan Reading)
	go func() {
		defer close(out)
		for r := range in {
			// check cancellation
			select {
			case <-ctx.Done():
				log.Printf("[filter] canceled")
				return
			default:
			}

			if r.Celsius < min || r.Celsius > max || r.Celsius < threshold {
				atomic.AddInt64(dropped, 1)
				log.Printf("[filter] dropped reading at: %s, value: %f", r.At, r.Celsius)
				continue
			}
			select {
			case out <- r:
			case <-ctx.Done():
				return
			}
		}
		log.Printf("[filter] input closed > closing out")
	}()
	return out
}

// ---- Stage 2: Transform (C→F) ----
func Transform(ctx context.Context, in <-chan Reading) <-chan Processed {
	out := make(chan Processed)
	go func() {
		defer close(out)
		for r := range in {
			select {
			case <-ctx.Done():
				log.Printf("[transform] canceled")
				return
			default:
			}
			f := r.Celsius*9/5 + 32
			p := Processed{Reading: r, Fahrenheit: f}
			select {
			case out <- p:
			case <-ctx.Done():
				return
			}
		}
		log.Printf("[transform] input closed > closing out")
	}()
	return out
}

// ---- Stage 3: Store (sink) ----
// thread-safe in-memory store + statistics
type Store struct {
	mu    sync.RWMutex
	data  []Processed
	count int64
}

func NewStore() *Store {
	return &Store{
		data: make([]Processed, 0, 1024),
	}
}

func (s *Store) Append(p Processed) {
	s.mu.Lock()
	s.data = append(s.data, p)
	s.mu.Unlock()
	atomic.AddInt64(&s.count, 1)
}

func (s *Store) Snapshot() []Processed {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]Processed, len(s.data))
	copy(cp, s.data)
	return cp
}

func (s *Store) Count() int64 { return atomic.LoadInt64(&s.count) }

// Consumer: reads from in until it is closed and stores the data
func Consume(ctx context.Context, in <-chan Processed, st *Store, wg *sync.WaitGroup) {
	defer wg.Done()
	for p := range in {
		select {
		case <-ctx.Done():
			log.Printf("[store] canceled")
			return
		default:
			st.Append(p)
		}
	}
	log.Printf("[store] input closed > done")
}

// ---- main: wire the pipeline ----
func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	rand.Seed(time.Now().UnixNano())

	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Stage0: generate 200 samples (for demo). For infinite: n<=0
	source := Sensor(parent, "sensor-A", 200, 200*time.Millisecond)

	var dropped int64
	// Stage1: only valid temperatures in [-50..100] and ≥ 30°C
	filt := Filter(parent, source, -50, 100, 30, &dropped)

	// Stage2: convert to Fahrenheit
	proc := Transform(parent, filt)

	// Stage3: consume and store
	store := NewStore()
	var wg sync.WaitGroup
	wg.Add(1)
	go Consume(parent, proc, store, &wg)

	// wait until consumer is done (when previous stages close their outputs)
	wg.Wait()

	// report
	totalKept := store.Count()
	totalDropped := atomic.LoadInt64(&dropped)
	fmt.Printf("\n=== REPORT ===\n")
	fmt.Printf("Kept:   %d readings (stored)\n", totalKept)
	fmt.Printf("Dropped:%d readings (filtered out)\n", totalDropped)

	// show a sample snapshot
	snap := store.Snapshot()
	if len(snap) > 0 {
		first := snap[0]
		last := snap[len(snap)-1]
		fmt.Printf("First kept: seq=%d, C=%.2f, F=%.2f\n", first.Sequence, first.Celsius, first.Fahrenheit)
		fmt.Printf("Last  kept: seq=%d, C=%.2f, F=%.2f\n", last.Sequence, last.Celsius, last.Fahrenheit)
	}
	log.Printf("[main] exit")
}
