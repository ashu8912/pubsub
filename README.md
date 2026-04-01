# Pub/Sub in Go

A from-scratch implementation of the **Publish-Subscribe** messaging pattern in Go, exposed over HTTP using Server-Sent Events (SSE).

---

## What is Pub/Sub?

Pub/Sub (Publish-Subscribe) is a messaging pattern where:

- **Publishers** send messages without knowing who will receive them.
- **Subscribers** register interest and receive all messages published after they subscribe.
- A **Broker** sits in the middle — it accepts published messages and fans them out to every active subscriber.

```
Publisher ──► Broker ──► Subscriber A
                   ├──► Subscriber B
                   └──► Subscriber C
```

The publisher and subscribers are fully **decoupled** — the publisher doesn't know (or care) how many subscribers exist.

---

## Project Structure

```
.
├── cmd/
│   └── main.go              # HTTP server (the real app)
├── internal/
│   └── broker/
│       └── broker.go         # Core Broker implementation
├── playground/
│   ├── main.go               # Standalone playground (channels basics)
│   ├── broker.go             # Self-contained broker demo (no HTTP)
│   └── fanout.go             # Fan-out pattern demo
├── go.mod
└── README.md
```

| Directory | Purpose |
|-----------|---------|
| `cmd/` | Entry point — HTTP server with `/publish` and `/subscribe` endpoints |
| `internal/broker/` | The reusable `Broker` struct used by the HTTP server |
| `playground/` | Standalone experiments to learn channels, fan-out, and pub/sub without HTTP |

---

## How the Broker Works (`internal/broker/broker.go`)

The `Broker` is the heart of the system. It has three operations:

### Data Structure

```go
type Broker struct {
    subscribers map[string]chan string   // serviceName → dedicated channel
    mu          sync.RWMutex            // protects concurrent access
}
```

Each subscriber gets its own **buffered channel** (capacity 5). The map key is the subscriber's `serviceName`.

### Subscribe

```go
func (b *Broker) Subscribe(serviceName string) (string, chan string)
```

1. Acquires a **write lock** (since we're mutating the map).
2. Creates a buffered channel `make(chan string, 5)`.
3. Stores it in the map under `serviceName`.
4. Returns the name and the channel so the caller can read from it.

### Publish

```go
func (b *Broker) Publish(msg string)
```

1. Acquires a **read lock** (multiple publishes can happen concurrently since we're only reading the map).
2. Iterates over every subscriber channel and sends the message to each one.
3. This is the **fan-out** — one message goes to all subscribers.

### Unsubscribe

```go
func (b *Broker) Unsubscribe(serviceName string)
```

1. Acquires a **write lock**.
2. Closes the subscriber's channel (signals the consumer goroutine to stop).
3. Deletes the entry from the map.

### Concurrency Model

- `sync.RWMutex` allows **multiple concurrent readers** (publishes) but **exclusive writers** (subscribe/unsubscribe).
- Buffered channels (`cap=5`) prevent the publisher from blocking if a subscriber is slightly slow to consume.

---

## HTTP Server (`cmd/main.go`)

The server exposes two endpoints:

### `POST /publish`

Publishes a message to all connected subscribers.

```bash
curl -X POST http://localhost:8080/publish \
  -H "Content-Type: application/json" \
  -d '{"message": "hello world"}'
```

- Accepts JSON body: `{"message": "your text"}`
- Calls `broker.Publish(msg)` which fans out to all subscribers.

### `GET /subscribe?serviceName=<name>`

Opens a persistent SSE (Server-Sent Events) connection. The server streams messages in real-time.

```bash
curl http://localhost:8080/subscribe?serviceName=serviceA
```

- Requires `serviceName` query parameter to identify the subscriber.
- Sets SSE headers (`Content-Type: text/event-stream`).
- Calls `broker.Subscribe(serviceName)` to get a channel.
- Loops forever reading from the channel and flushing each message to the HTTP response.
- When the client disconnects, `r.Context().Done()` fires, and the handler calls `broker.Unsubscribe()` to clean up.

### Request Flow

```
Client A (curl subscribe)  ──► GET /subscribe?serviceName=A
                                  │
                                  ▼
                              broker.Subscribe("A")
                              returns chan string
                                  │
                                  ▼
                              blocks on <-ch, waiting for messages
                                  │
Client B (curl publish)     ──► POST /publish {"message":"hi"}
                                  │
                                  ▼
                              broker.Publish("hi")
                                  │
                                  ▼
                              sends "hi" to every subscriber's channel
                                  │
                                  ▼
Client A receives:           data: hi
```

---

## Playground (`playground/`)

Standalone experiments to understand the building blocks before the full HTTP version.

### `playground/main.go` — Basic Channels

Simple producer-consumer with a single channel. One goroutine sends, the main goroutine receives via `range`.

### `playground/fanout.go` — Fan-Out Pattern

One producer channel, multiple consumer goroutines. Each value from the channel goes to **exactly one** consumer (work distribution, not broadcast). This is **different** from pub/sub where every subscriber gets every message.

### `playground/broker.go` — Full Pub/Sub (No HTTP)

A self-contained version of the broker with UUID-based subscriber IDs. Three consumers subscribe, messages are published, then all are unsubscribed. Demonstrates:
- Each consumer gets **every** message (true fan-out/broadcast).
- `sync.WaitGroup` ensures the program waits for all consumers to finish.
- Unsubscribing closes channels, which terminates the `for range` loops in goroutines.

---

## Key Go Concepts Used

| Concept | Where | Why |
|---------|-------|-----|
| **Buffered channels** | `make(chan string, 5)` | Decouple publisher speed from subscriber speed |
| **Goroutines** | Subscriber consumers | Each subscriber reads from its channel concurrently |
| **`sync.RWMutex`** | Broker struct | Safe concurrent read (publish) / write (subscribe/unsubscribe) |
| **`sync.WaitGroup`** | Playground | Wait for all goroutines to finish before exiting |
| **`http.Flusher`** | SSE streaming | Push data to the client immediately without buffering |
| **`r.Context().Done()`** | Subscribe handler | Detect client disconnect to clean up |
| **Server-Sent Events** | `/subscribe` endpoint | One-way real-time streaming from server to client |

---

## Fan-Out vs Fan-In vs Pub/Sub

| Pattern | Behavior |
|---------|----------|
| **Fan-Out** (playground/fanout.go) | One channel, multiple consumers — each message goes to **one** consumer |
| **Fan-In** | Multiple producers, one consumer channel |
| **Pub/Sub** (this project) | One message is **broadcast** to **all** subscribers via their own channels |

---

## Running

```bash
# Start the server
go run cmd/main.go

# In another terminal — subscribe
curl http://localhost:8080/subscribe?serviceName=serviceA

# In another terminal — publish
curl -X POST http://localhost:8080/publish \
  -H "Content-Type: application/json" \
  -d '{"message": "hello from pubsub"}'

# The subscribe terminal will print:
# data: hello from pubsub
```

```bash
# Run the playground standalone demo
go run playground/broker.go playground/fanout.go playground/main.go
```
