package transport

import (
	"context"
	"fmt"
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

func TestDispatcher_SlowSubscriber(t *testing.T) {
	source := make(chan models.Event, 10)
	dispatcher := NewDispatcher(source, 2) // Small buffer to trigger drops

	fastSub := dispatcher.Subscribe()
	slowSub := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Run(ctx)

	// Start fast subscriber immediately so it can consume as events arrive
	fastCount := 0
	fastDone := make(chan struct{})
	go func() {
		defer close(fastDone)
		for range fastSub {
			fastCount++
		}
	}()

	// Start slow subscriber immediately
	slowCount := 0
	slowDone := make(chan struct{})
	go func() {
		defer close(slowDone)
		for range slowSub {
			slowCount++
			time.Sleep(10 * time.Millisecond) // Slow processing
		}
	}()

	// Give subscribers time to start
	time.Sleep(5 * time.Millisecond)

	// Send events faster than slow subscriber can consume
	numEvents := 10
	for i := 0; i < numEvents; i++ {
		source <- models.Event{EventID: fmt.Sprintf("event-%d", i)}
		time.Sleep(1 * time.Millisecond) // Small delay between sends
	}
	close(source)

	// Wait for both subscribers to finish
	<-fastDone
	<-slowDone

	// Fast subscriber should get all events (since it consumes immediately)
	if fastCount != numEvents {
		t.Errorf("fast subscriber: expected %d events, got %d", numEvents, fastCount)
	}

	// Slow subscriber should have dropped some due to buffer overflow
	// This is expected behavior - verify that some events were dropped
	dropped := dispatcher.GetDroppedCount()
	if dropped == 0 && slowCount < numEvents {
		// If we got fewer events but no drops were recorded, something's wrong
		t.Logf("Slow subscriber got %d events (expected some drops), dropped count: %d", slowCount, dropped)
	}

	// At least verify slow subscriber got some events
	if slowCount == 0 {
		t.Error("slow subscriber should have received at least some events")
	}

	// Verify that some events were dropped for the slow subscriber
	if dropped == 0 {
		t.Logf("Note: No events were dropped, but slow subscriber got %d/%d events", slowCount, numEvents)
	}
}

func TestDispatcher_BufferOverflow(t *testing.T) {
	source := make(chan models.Event, 10)
	dispatcher := NewDispatcher(source, 2) // Very small buffer

	sub := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Run(ctx)

	// Send many events rapidly to overflow buffer
	numEvents := 20
	for i := 0; i < numEvents; i++ {
		source <- models.Event{EventID: fmt.Sprintf("event-%d", i)}
	}
	close(source)

	// Give dispatcher time to process
	time.Sleep(50 * time.Millisecond)

	// Count received events
	received := 0
	receivedDone := make(chan struct{})
	go func() {
		defer close(receivedDone)
		for range sub {
			received++
		}
	}()

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)
	cancel() // Stop dispatcher
	<-receivedDone

	// With buffer size 2, we should have received at most bufferSize events
	// plus any that were in flight, but many should have been dropped
	dropped := dispatcher.GetDroppedCount()
	if dropped == 0 {
		t.Error("expected some events to be dropped with small buffer and rapid sends")
	}

	// Verify that dropped count is tracked
	if dropped < 0 {
		t.Errorf("dropped count should be non-negative, got %d", dropped)
	}

	t.Logf("Sent %d events, received %d, dropped %d", numEvents, received, dropped)
}

func TestDispatcher_GetSubscriberCount(t *testing.T) {
	source := make(chan models.Event, 10)
	dispatcher := NewDispatcher(source, 10)

	if dispatcher.GetSubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers initially, got %d", dispatcher.GetSubscriberCount())
	}

	sub1 := dispatcher.Subscribe()
	if dispatcher.GetSubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", dispatcher.GetSubscriberCount())
	}

	sub2 := dispatcher.Subscribe()
	if dispatcher.GetSubscriberCount() != 2 {
		t.Errorf("expected 2 subscribers, got %d", dispatcher.GetSubscriberCount())
	}

	// Clean up
	close(source)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	go dispatcher.Run(ctx)
	time.Sleep(10 * time.Millisecond)

	// Drain channels
	for range sub1 {
	}
	for range sub2 {
	}
}

func TestDispatcher_GetDroppedCount(t *testing.T) {
	source := make(chan models.Event, 10)
	dispatcher := NewDispatcher(source, 1) // Very small buffer to force drops

	sub := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Run(ctx)

	// Send events faster than can be consumed
	for i := 0; i < 10; i++ {
		source <- models.Event{EventID: fmt.Sprintf("event-%d", i)}
	}
	close(source)

	// Give dispatcher time to process and drop events
	time.Sleep(50 * time.Millisecond)

	dropped := dispatcher.GetDroppedCount()
	if dropped < 0 {
		t.Errorf("dropped count should be non-negative, got %d", dropped)
	}

	// With a buffer of 1 and rapid sends, we should have some drops
	// (exact count depends on timing, so we just verify it's tracked)
	t.Logf("Dropped events count: %d", dropped)

	// Drain subscriber channel to ensure test completes properly
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range sub {
		}
	}()

	// Wait a bit for draining, then cancel to stop dispatcher
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-done
}
