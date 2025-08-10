package main

func requireNonEmpty(xs []float64) error {
	if len(xs) == 0 {
		return ErrEmptySlice
	}
	return nil
}
