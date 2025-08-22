package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gloonch/scenarios/P06/pricecache"
	"github.com/redis/go-redis/v9"
)

func main() {

	var (
		redisAddr = flag.String("redis", "localhost:6379", "Redis address")
		symbol    = flag.String("symbol", "BTC", "Optional symbol to read from cache periodically")
		interval  = flag.Duration("interval", 2*time.Second, "How often to read from cache")
		channel   = flag.String("channel", "prices", "Redis pub/sub channel")
	)

	flag.Parse()

	redis := redis.NewClient(&redis.Options{Addr: *redisAddr})
	defer redis.Close()

	priceCache := pricecache.New(redis, 0, *channel) // no TTL, just channel

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. pub/sub listener
	go func() {
		err := priceCache.Subscribe(ctx, func(p pricecache.Price) error {
			log.Printf("[sub] LIVE %s=%0.2f @ %s", p.Symbol, p.Price, p.At.Format(time.Kitchen))
			return nil
		})
		if err != nil && err != context.Canceled {
			log.Printf("[sub] subscribe error: %v", err)
		}
	}()

	// 2. periodic cache reader (to observe TTL expiry)
	t := time.NewTicker(*interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[sub] stopping: %v", ctx.Err())
			return
		case <-t.C:
			p, err := priceCache.GetPrice(ctx, *symbol)
			if err != nil {
				log.Printf("[sub] CACHE miss for %s (maybe TTL expired)\n", *symbol)
				continue
			}
			log.Printf("[sub] CACHED %s=%0.2f @ %s\n", p.Symbol, p.Price, p.At.Format(time.Kitchen))
		}
	}
}
