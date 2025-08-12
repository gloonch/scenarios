package main

import (
	"math/rand"
	"strconv"
	"testing"
)

func Benchmark_ReadHeavy_Mutex(b *testing.B) {
	c := NewCounterM()
	paths := makeTestPaths(1000)

	for _, p := range paths {
		c.Inc(p)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := paths[i%len(paths)]
		_ = c.Get(p)   // 90% read
		if i%10 == 0 { // 10% write
			c.Inc(p)
		}
	}
}

func Benchmark_ReadHeavy_RWMutex(b *testing.B) {
	c := NewCounterRW()
	paths := makeTestPaths(1000)
	for _, p := range paths {
		c.Inc(p)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := paths[i%len(paths)]
		_ = c.Get(p)
		if i%10 == 0 {
			c.Inc(p)
		}
	}
}

func Benchmark_WriteHeavy_Mutex(b *testing.B) {
	c := NewCounterM()
	p := "/hot"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc(p) // write-only
	}
}

func Benchmark_WriteHeavy_RWMutex(b *testing.B) {
	c := NewCounterRW()
	p := "/hot"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc(p) // write-only
	}
}

func makeTestPaths(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "/p/" + strconv.Itoa(rand.Intn(n))
	}
	return out
}
