package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	dir := "P02/tmp/files/"
	log.Printf("[main] start. scanning dir=%q", dir)

	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		log.Fatalf("[main] glob error: %v", err)
	}
	if len(files) == 0 {
		log.Printf("[main] no files found in %q (nothing to do)", dir)
		return
	}
	log.Printf("[main] found %d files", len(files))

	var (
		totalSize int64
		mu        sync.Mutex
		wg        sync.WaitGroup
		nextID    int64
	)

	for _, f := range files {
		id := atomic.AddInt64(&nextID, 1)
		path := f

		log.Printf("[main] wg.Add(1) for job=%d file=%s", id, path)
		wg.Add(1)

		log.Printf("[main] launching goroutine for job=%d", id)
		go func(jobID int64, p string) {
			defer func() {
				log.Printf("[job %d] DONE → calling wg.Done()", jobID)
				wg.Done()
			}()

			log.Printf("[job %d] START processing file=%s", jobID, p)

			start := time.Now()
			size, err := getFileSize(jobID, p)
			if err != nil {
				log.Printf("[job %d] error reading %s: %v", jobID, p, err)
				return
			}
			elapsed := time.Since(start)

			log.Printf("[job %d] got size=%dB in %s → acquiring lock", jobID, size, elapsed)

			mu.Lock()
			log.Printf("[job %d] LOCKED. total(before)=%d", jobID, totalSize)
			totalSize += size
			log.Printf("[job %d] updated total(after)=%d → unlocking", jobID, totalSize)
			mu.Unlock()

			log.Printf("[job %d] UNLOCKED. file=%s, size=%dB", jobID, p, size)
		}(id, path)
	}

	log.Printf("[main] all goroutines launched → entering wg.Wait() (blocking)")
	wg.Wait()
	log.Printf("[main] wg.Wait() returned (all jobs done)")

	fmt.Printf("Total size: %d bytes\n", totalSize)
	log.Printf("[main] exit")
}

func getFileSize(jobID int64, path string) (int64, error) {
	log.Printf("[job %d] opening file=%s", jobID, path)
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = f.Close()
		log.Printf("[job %d] closed file=%s", jobID, path)
	}()

	// time.Sleep(50 * time.Millisecond)

	n, err := io.Copy(io.Discard, f)
	log.Printf("[job %d] io.Copy done bytes=%d err=%v", jobID, n, err)
	return n, err
}
