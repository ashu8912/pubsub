package main

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type Broker struct {
	subscriber map[string]chan string
	mu         sync.RWMutex
}

func NewBroker() *Broker {
	return &Broker{
		subscriber: make(map[string]chan string),
		mu:         sync.RWMutex{},
	}
}

func (b *Broker) GenerateSubID() string {
	return uuid.New().String()
}

func (b *Broker) Subscribe() (string, chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.GenerateSubID()
	bufCh := make(chan string, 5)
	b.subscriber[id] = bufCh

	return id, bufCh
}

func (b *Broker) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch, ok := b.subscriber[id]
	if !ok {
		return
	}
	close(ch)
	delete(b.subscriber, id)
}

func (b *Broker) Publish(msg string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscriber {
		ch <- msg
	}
}

func main() {
	broker := NewBroker()
	var wg sync.WaitGroup
	consumers := 3

	ids := make([]string, 0, consumers)
	for i := 0; i < consumers; i++ {
		id, ch := broker.Subscribe()
		wg.Add(1)
		ids = append(ids, id)

		go func() {
			defer wg.Done()
			for val := range ch {
				fmt.Printf("Consumer %s got value %s \n", id, val)
			}
		}()

	}

	broker.Publish("hello")
	broker.Publish("world")
	broker.Publish("folks")
	broker.Publish("We are learning pubsub")

	for _, id := range ids {
		broker.Unsubscribe(id)
	}
	wg.Wait()
}
