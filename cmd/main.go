package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ashu8912/pubsub/internal/broker"
)

type Message struct {
	Message string `json:"message"`
}

func main() {
	broker := broker.NewBroker()
	http.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only post allowed", http.StatusMethodNotAllowed)
			return
		}
		var msg Message

		err := json.NewDecoder(r.Body).Decode(&msg)
		fmt.Println("Publishing message ", msg.Message)

		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if msg.Message == "" {
			http.Error(w, "Missing Fields message is required ", http.StatusBadRequest)
			return
		}

		broker.Publish(msg.Message)
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		fmt.Println("New subscriber connected")
		serviceName := r.URL.Query().Get("serviceName")
		if serviceName == "" {
			http.Error(w, "Missing serviceName query parameter", http.StatusBadRequest)
			return
		}

		_, ch := broker.Subscribe(serviceName)
		defer broker.Unsubscribe(serviceName)
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		for {
			select {
			case msg := <-ch:
				fmt.Fprintf(w, "data: %s\n\n", msg)
				flusher.Flush()
			case <-r.Context().Done():
				fmt.Println("Subscriber disconnected:", serviceName)
				return
			}

		}

	})
	fmt.Println("Listening on port 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
