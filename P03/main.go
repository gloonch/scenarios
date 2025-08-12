package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

func main() {
	c := NewCounterRW()
	//c := NewCounterM()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ { // 8 simultaneous users
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			path := fmt.Sprintf("/post/%d", id%3) // 3 paths
			for j := 0; j < 1000; j++ {
				c.Inc(path)
				_ = c.Get(path)
				time.Sleep(1 * time.Millisecond)
			}
			log.Printf("[u%d] done", id)
		}(i)
	}
	wg.Wait()

	fmt.Println("Snapshot:", c.Snapshot())
}
