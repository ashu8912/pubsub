package broker

import "sync"

type Broker struct {
	subscribers map[string]chan string
	mu          sync.RWMutex
}

func (b *Broker) Subscribe(serviceName string) (string, chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	bufCh := make(chan string, 5)
	b.subscribers[serviceName] = bufCh

	return serviceName, bufCh
}

func (b *Broker) Unsubscribe(serviceName string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch, ok := b.subscribers[serviceName]
	if !ok {
		return
	}
	close(ch)
	delete(b.subscribers, serviceName)
}

func (b *Broker) Publish(msg string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers {
		ch <- msg
	}
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string]chan string),
		mu:          sync.RWMutex{},
	}
}
