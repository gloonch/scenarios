package main

import "sync"

type CounterRW struct {
	mu sync.RWMutex
	m  map[string]int64
}

func NewCounterRW() *CounterRW {
	return &CounterRW{m: make(map[string]int64)}
}

func (c *CounterRW) Inc(path string) {
	c.mu.Lock()
	c.m[path]++
	c.mu.Unlock()
}

func (c *CounterRW) Get(path string) int64 {
	c.mu.RLock()
	v := c.m[path]
	c.mu.RUnlock()
	return v
}

func (c *CounterRW) Snapshot() map[string]int64 {
	c.mu.RLock()
	cp := make(map[string]int64, len(c.m))
	for k, v := range c.m {
		cp[k] = v
	}
	c.mu.RUnlock()
	return cp
}
