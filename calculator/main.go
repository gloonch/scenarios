package main

import (
	"fmt"
)

func main() {
	nums := []float64{1, 2, 3, 4, 5}
	mean, _ := Mean(nums)
	fmt.Println("Mean:", mean) // 3
}
