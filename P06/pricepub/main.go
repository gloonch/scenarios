package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gloonch/scenarios/P06/pricecache"
	"github.com/redis/go-redis/v9"
)

func main() {
	var (
		redisAddr = flag.String("redis", "localhost:6379", "Redis address")
		symbols   = flag.String("symbols", "BTC,ETH,ADA", "Comma separated symbols")
		rate      = flag.Duration("rate", 500*time.Millisecond, "Update rate per symbol")
		ttl       = flag.Duration("ttl", 5*time.Second, "Redis TTL for a price")
		channel   = flag.String("ch", "prices", "Redis Pub/Sub channel")
	)
	flag.Parse()

	rdb := redis.NewClient(&redis.Options{Addr: *redisAddr})
	defer rdb.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pc := pricecache.New(rdb, *ttl, *channel)
	syms := splitAndTrim(*symbols)

	log.Printf("[pub] start. symbols=%v rate=%s ttl=%s channel=%s", syms, *rate, *ttl, *channel)

	ticks := make([]*time.Ticker, len(syms))
	for i := range syms {
		ticks[i] = time.NewTicker(*rate)
		defer ticks[i].Stop()
	}

	// last prices map to create a small random walk per symbol
	last := make(map[string]float64)
	for _, s := range syms {
		last[s] = 100 + rand.Float64()*50
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("[pub] stopping: %v", ctx.Err())
			return
		default:
			for i, s := range syms {
				select {
				case <-ticks[i].C:
					// random walk
					last[s] += (rand.Float64() - 0.5) * 2.0 // move ±1.0
					p := pricecache.Price{Symbol: s, Price: round(last[s], 2), At: time.Now()}
					if err := pc.UpsertPrice(ctx, p); err != nil {
						log.Printf("[pub] upsert error for %s: %v", s, err)
					} else {
						log.Printf("[pub] %s=%0.2f (TTL=%s)", s, p.Price, ttl.String())
					}
				default:
				}
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, strings.ToUpper(p))
		}
	}
	return out
}

func round(v float64, n int) float64 {
	p := 1.0
	for i := 0; i < n; i++ {
		p *= 10
	}
	return float64(int(v*p+0.5)) / p
}
