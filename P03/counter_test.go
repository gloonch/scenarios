package main

import (
	"sync"
	"testing"
)

func runConcurrent(t *testing.T, inc func(string), get func(string) int64) {
	var wg sync.WaitGroup
	path := "/home"

	// 100 write , 100 read parallel
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			inc(path)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = get(path)
		}()
	}
	wg.Wait()
}

func TestCounterM(t *testing.T) {
	c := NewCounterM()
	runConcurrent(t, c.Inc, c.Get)
	if got := c.Get("/home"); got <= 0 {
		t.Fatalf("want > 0, got %d", got)
	}
	if snap := c.Snapshot(); len(snap) == 0 {
		t.Fatalf("snapshot empty")
	}
}

func TestCounterRW(t *testing.T) {
	c := NewCounterRW()
	runConcurrent(t, c.Inc, c.Get)
	if got := c.Get("/home"); got <= 0 {
		t.Fatalf("want > 0, got %d", got)
	}
	if snap := c.Snapshot(); len(snap) == 0 {
		t.Fatalf("snapshot empty")
	}
}
