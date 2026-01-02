package transport

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/synheart/synheart-cli/internal/models"
)

func TestDispatcher_SingleSubscriber(t *testing.T) {
	source := make(chan models.Event, 10)
	dispatcher := NewDispatcher(source, 10)
	subscriber := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Run(ctx)

	for i := 0; i < 5; i++ {
		source <- models.Event{EventID: string(rune('A' + i))}
	}
	close(source)

	time.Sleep(10 * time.Millisecond)

	count := 0
	for range subscriber {
		count++
	}

	if count != 5 {
		t.Errorf("expected 5 events, got %d", count)
	}
}

func TestDispatcher_MultipleSubscribers(t *testing.T) {
	source := make(chan models.Event, 10)
	dispatcher := NewDispatcher(source, 10)

	sub1 := dispatcher.Subscribe()
	sub2 := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Run(ctx)

	numEvents := 10
	for i := 0; i < numEvents; i++ {
		source <- models.Event{EventID: string(rune('A' + i))}
	}
	close(source)

	time.Sleep(10 * time.Millisecond)

	var wg sync.WaitGroup
	var count1, count2 int

	wg.Add(2)
	go func() {
		defer wg.Done()
		for range sub1 {
			count1++
		}
	}()
	go func() {
		defer wg.Done()
		for range sub2 {
			count2++
		}
	}()
	wg.Wait()

	if count1 != numEvents {
		t.Errorf("subscriber 1: expected %d events, got %d", numEvents, count1)
	}
	if count2 != numEvents {
		t.Errorf("subscriber 2: expected %d events, got %d", numEvents, count2)
	}
}

func TestDispatcher_SubscribersReceiveSameEvents(t *testing.T) {
	source := make(chan models.Event, 10)
	dispatcher := NewDispatcher(source, 10)

	sub1 := dispatcher.Subscribe()
	sub2 := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Run(ctx)

	events := []models.Event{
		{EventID: "event-1"},
		{EventID: "event-2"},
		{EventID: "event-3"},
	}
	for _, e := range events {
		source <- e
	}
	close(source)

	time.Sleep(10 * time.Millisecond)

	var received1, received2 []string
	for e := range sub1 {
		received1 = append(received1, e.EventID)
	}
	for e := range sub2 {
		received2 = append(received2, e.EventID)
	}

	for i, e := range events {
		if received1[i] != e.EventID {
			t.Errorf("sub1 event %d: got %s, want %s", i, received1[i], e.EventID)
		}
		if received2[i] != e.EventID {
			t.Errorf("sub2 event %d: got %s, want %s", i, received2[i], e.EventID)
		}
	}
}

func TestDispatcher_ContextCancellation(t *testing.T) {
	source := make(chan models.Event, 10)
	dispatcher := NewDispatcher(source, 10)

	sub := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		dispatcher.Run(ctx)
		close(done)
	}()

	source <- models.Event{EventID: "before-cancel"}
	time.Sleep(5 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Error("dispatcher did not stop after context cancellation")
	}

	// Subscriber channel should be closed
	_, ok := <-sub
	if ok {
		// First event might still be there
		_, ok = <-sub
	}
	if ok {
		t.Error("subscriber channel should be closed after dispatcher stops")
	}
}
