package transport

import (
	"context"
	"sync"

	"github.com/synheart/synheart-cli/internal/models"
)

// Dispatcher copies events from one source to multiple subscribers
type Dispatcher struct {
	source      <-chan models.Event
	subscribers []chan models.Event
	bufferSize  int
	mu          sync.Mutex
}

func NewDispatcher(source <-chan models.Event, bufferSize int) *Dispatcher {
	return &Dispatcher{
		source:      source,
		subscribers: make([]chan models.Event, 0),
		bufferSize:  bufferSize,
	}
}

// Subscribe returns a channel that receives copies of all source events
func (d *Dispatcher) Subscribe() <-chan models.Event {
	ch := make(chan models.Event, d.bufferSize)
	d.mu.Lock()
	d.subscribers = append(d.subscribers, ch)
	d.mu.Unlock()
	return ch
}

// Run blocks until ctx is cancelled or source closes
func (d *Dispatcher) Run(ctx context.Context) {
	defer d.closeSubscribers()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-d.source:
			if !ok {
				return
			}
			d.dispatch(event, ctx)
		}
	}
}

func (d *Dispatcher) dispatch(event models.Event, ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, sub := range d.subscribers {
		select {
		case sub <- event:
		case <-ctx.Done():
			return
		default:
			// buffer full, drop to keep generator running
		}
	}
}

func (d *Dispatcher) closeSubscribers() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, sub := range d.subscribers {
		close(sub)
	}
}
