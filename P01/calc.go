package main

import (
	"math"
	"sort"
)

func Sum(xs []float64) (float64, error) {
	if err := requireNonEmpty(xs); err != nil {
		return 0, err
	}
	var s float64
	for _, v := range xs {
		s += v
	}
	return s, nil
}

func Mean(xs []float64) (float64, error) {
	if err := requireNonEmpty(xs); err != nil {
		return 0, err
	}
	s, _ := Sum(xs)
	return s / float64(len(xs)), nil
}

func Median(xs []float64) (float64, error) {
	if err := requireNonEmpty(xs); err != nil {
		return 0, err
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 { // odd
		return cp[n/2], nil
	}
	return (cp[n/2-1] + cp[n/2]) / 2, nil
}

func Variance(xs []float64) (float64, error) {
	if err := requireNonEmpty(xs); err != nil {
		return 0, err
	}
	m, _ := Mean(xs)
	var acc float64
	for _, v := range xs {
		d := v - m
		acc += d * d
	}
	return acc / float64(len(xs)), nil
}

func Std(xs []float64) (float64, error) {
	v, err := Variance(xs)
	if err != nil {
		return 0, err
	}
	return math.Sqrt(v), nil
}
