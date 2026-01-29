package transport

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
)

// Dispatcher copies payloads from one source to multiple subscribers.
// When a subscriber's buffer is full, events are dropped to prevent blocking
// the generator. Dropped events are logged and counted for monitoring.
type Dispatcher struct {
	source       <-chan []byte
	subscribers  []chan []byte
	bufferSize   int
	mu           sync.Mutex
	droppedTotal int64 // atomic counter for total dropped events
}

func NewDispatcher(source <-chan []byte, bufferSize int) *Dispatcher {
	return &Dispatcher{
		source:      source,
		subscribers: make([]chan []byte, 0),
		bufferSize:  bufferSize,
	}
}

// Subscribe returns a channel that receives copies of all source events.
// Each subscriber gets its own buffered channel with the configured buffer size.
// Subscribers should be added before calling Run() to ensure they receive all events.
func (d *Dispatcher) Subscribe() <-chan []byte {
	ch := make(chan []byte, d.bufferSize)
	d.mu.Lock()
	d.subscribers = append(d.subscribers, ch)
	d.mu.Unlock()
	return ch
}

// GetSubscriberCount returns the current number of active subscribers.
// This is useful for monitoring and debugging.
func (d *Dispatcher) GetSubscriberCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.subscribers)
}

// GetDroppedCount returns the total number of events that were dropped
// due to subscriber buffers being full. This counter is thread-safe.
func (d *Dispatcher) GetDroppedCount() int64 {
	return atomic.LoadInt64(&d.droppedTotal)
}

// Run blocks until ctx is cancelled or source closes
func (d *Dispatcher) Run(ctx context.Context) {
	defer d.closeSubscribers()

	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-d.source:
			if !ok {
				return
			}
			d.dispatch(data, ctx)
		}
	}
}

func (d *Dispatcher) dispatch(data []byte, ctx context.Context) {
	d.mu.Lock()
	subs := d.subscribers // Copy slice reference to minimize lock time
	d.mu.Unlock()

	dropped := 0
	for _, sub := range subs {
		select {
		case sub <- data:
			// Successfully sent
		case <-ctx.Done():
			return
		default:
			// Buffer full - drop event to prevent blocking generator
			dropped++
			atomic.AddInt64(&d.droppedTotal, 1)
		}
	}

	// Log dropped events (only if any were dropped to avoid log spam)
	if dropped > 0 {
		log.Printf("Dispatcher: dropped packet for %d subscriber(s) (buffer full)", dropped)
	}
}

func (d *Dispatcher) closeSubscribers() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, sub := range d.subscribers {
		close(sub)
	}
}
