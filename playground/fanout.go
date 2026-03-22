package main

import (
	"fmt"
	"sync"
)

func fanout() {
	bufCh := make(chan int, 10)
	var wg sync.WaitGroup
	go func() {
		for i := 1; i <= 20; i++ {
			bufCh <- i
		}
		close(bufCh)
	}()

	numConsumers := 3

	for i := 0; i < numConsumers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for val := range bufCh {
				fmt.Printf("Consumer %d got: %d \n", i, val)
			}
		}(i)
	}

	wg.Wait()
}
