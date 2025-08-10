package main

import "testing"

func TestSum(t *testing.T) {
	tests := []struct {
		name    string
		in      []float64
		want    float64
		wantErr bool
	}{
		{"normal", []float64{1, 2, 3}, 6, false},
		{"single", []float64{5}, 5, false},
		{"empty", []float64{}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Sum(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("got=%v, want=%v", got, tt.want)
			}
		})
	}
}

func TestMeanMedianStd(t *testing.T) {
	mean, _ := Mean([]float64{1, 2, 3, 4}) // 2.5
	if mean != 2.5 {
		t.Fatalf("mean got=%v", mean)
	}

	medOdd, _ := Median([]float64{9, 1, 5}) // 5
	if medOdd != 5 {
		t.Fatalf("median odd got=%v", medOdd)
	}

	medEven, _ := Median([]float64{1, 2, 3, 4}) // 2.5
	if medEven != 2.5 {
		t.Fatalf("median even got=%v", medEven)
	}

	std, _ := Std([]float64{2, 2, 2, 2}) // 0
	if std != 0 {
		t.Fatalf("std got=%v", std)
	}
}
