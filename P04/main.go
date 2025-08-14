package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type Result struct {
	URL        string
	Bytes      int64
	Err        error
	Elapsed    time.Duration
	StatusCode int
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	urls := []string{
		"https://mobile.ir",
		"https://httpbin.org/delay/2",
		"https://httpbin.org/delay/5",
	}

	parent, cancelAll := context.WithCancel(context.Background())
	defer cancelAll()

	results := make(chan Result)
	var wg sync.WaitGroup

	log.Printf("[main] launching %d downloads", len(urls))
	for _, u := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			res := fetchWithTimeout(parent, url, 3*time.Second)
			select {
			case results <- res:
			case <-parent.Done():
			}
		}(u)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var ok, failed int
	var totalBytes int64

	log.Printf("[main] waiting for results (selecting on results channel)")
	for r := range results {
		if r.Err != nil {
			log.Printf("[result] URL=%s ERR=%v elapsed=%s", r.URL, r.Err, r.Elapsed)
			failed++
			// if wanted to cancel all with the first failure
			// cancelAll()
			continue
		}
		log.Printf("[result] URL=%s status=%d bytes=%d elapsed=%s",
			r.URL, r.StatusCode, r.Bytes, r.Elapsed)
		ok++
		totalBytes += r.Bytes
	}

	log.Printf("[main] done. ok=%d failed=%d totalBytes=%d", ok, failed, totalBytes)
}

func fetchWithTimeout(parent context.Context, url string, perReq time.Duration) Result {
	ctx, cancel := context.WithTimeout(parent, perReq)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{URL: url, Err: err}
	}

	client := &http.Client{
		Timeout: 0,
	}

	log.Printf("[fetch] start url=%s (deadline=%s)", url, start.Add(perReq).Format(time.RFC3339Nano))

	resp, err := client.Do(req)
	if err != nil {
		return Result{URL: url, Err: err, Elapsed: time.Since(start)}
	}
	defer resp.Body.Close()

	n, copyErr := io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(start)

	select {
	case <-ctx.Done():
		return Result{URL: url, Err: ctx.Err(), Elapsed: elapsed, StatusCode: resp.StatusCode}
	default:
		// no problem
	}

	if copyErr != nil {
		return Result{URL: url, Err: copyErr, Elapsed: elapsed, StatusCode: resp.StatusCode}
	}
	return Result{
		URL:        url,
		StatusCode: resp.StatusCode,
		Bytes:      n,
		Elapsed:    elapsed,
	}
}
