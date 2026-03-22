package main

import "fmt"

func example() {
	ch := make(chan string)

	go func() {
		ch <- "hello"
		ch <- "world"
		ch <- "done"
		close(ch)
	}()

	for val := range ch {
		fmt.Println(val)
	}
}
