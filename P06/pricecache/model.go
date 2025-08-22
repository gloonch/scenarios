package pricecache

import "time"

type Price struct {
	Symbol string	`json:"symbol"` // e.g. "BTC"
	Price  float64	`json:"price"` // e.g. 10000.00
	At     time.Time	`json:"at"` // e.g. 2021-01-01T00:00:00Z
}

