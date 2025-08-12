package main

import "sync"

type CounterM struct {
	mu sync.Mutex
	m  map[string]int64
}

func NewCounterM() *CounterM {
	return &CounterM{m: make(map[string]int64)}
}

func (c *CounterM) Inc(path string) {
	c.mu.Lock()
	c.m[path]++
	c.mu.Unlock()
}

func (c *CounterM) Get(path string) int64 {
	c.mu.Lock()
	v := c.m[path]
	c.mu.Unlock()
	return v
}

func (c *CounterM) Snapshot() map[string]int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make(map[string]int64, len(c.m))
	for k, v := range c.m {
		cp[k] = v
	}
	return cp
}
