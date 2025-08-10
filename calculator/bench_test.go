package main

import (
	"math/rand"
	"testing"
)

func benchSlice(n int) []float64 {
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = rand.Float64()
	}
	return xs
}

func BenchmarkMean_1e3(b *testing.B) {
	data := benchSlice(1_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Mean(data)
	}
}

func BenchmarkMedian_1e3(b *testing.B) {
	data := benchSlice(1_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Median(data)
	}
}
