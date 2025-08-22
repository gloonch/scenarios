package pricecache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	redis   *redis.Client
	ttl     time.Duration
	channel string
}

func New(redis *redis.Client, ttl time.Duration, channel string) *Cache {
	return &Cache{redis: redis, ttl: ttl, channel: channel}
}

func keyFor(symbol string) string {
	return fmt.Sprintf("price:%s", strings.ToUpper(symbol))
}

// UpsertPrice sets the price in Redis with TTL and publishes an event via Pub/Sub.
// Uses TxPipeline to ensure atomically Set + Publish.
func (c *Cache) UpsertPrice(ctx context.Context, p Price) error {
	payload, err := json.Marshal(p)
	if err != nil {
		return err
	}

	_, err = c.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, keyFor(p.Symbol), payload, c.ttl)
		pipe.Publish(ctx, c.channel, payload)
		return nil
	})
	return err
}

// GetPrice reads the last chaced price from Redis.
func (c *Cache) GetPrice(ctx context.Context, symbol string) (Price, error) {
	val, err := c.redis.Get(ctx, keyFor(symbol)).Result()
	if err == redis.Nil {
		return Price{}, ErrNotFound
	}
	if err != nil {
		return Price{}, err
	}

	var p Price
	if err := json.Unmarshal([]byte(val), &p); err != nil {
		return Price{}, err
	}
	return p, nil
}

// Subscribe starts a Pub/Sub subscriptions. It calls handle(p) for each message.
// Blocks until ctx is done or PubSub returns an error.
func (c *Cache) Subscribe(ctx context.Context, handle func(Price) error) error {
	sub := c.redis.Subscribe(ctx, c.channel)
	defer sub.Close()

	ch := sub.Channel() // returns <- chan *redis.Message
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil // channel closed
			}
			var p Price
			if err := json.Unmarshal([]byte(msg.Payload), &p); err != nil {
				// ignore invalid messages
				continue
			}
			handle(p)
		}
	}
}
