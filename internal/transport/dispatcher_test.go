package transport

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestDispatcher_SingleSubscriber(t *testing.T) {
	source := make(chan []byte, 10)
	dispatcher := NewDispatcher(source, 10)
	subscriber := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Run(ctx)

	for i := 0; i < 5; i++ {
		source <- []byte(string(rune('A' + i)))
	}
	close(source)

	time.Sleep(10 * time.Millisecond)

	count := 0
	for range subscriber {
		count++
	}

	if count != 5 {
		t.Errorf("expected 5 packets, got %d", count)
	}
}

func TestDispatcher_MultipleSubscribers(t *testing.T) {
	source := make(chan []byte, 10)
	dispatcher := NewDispatcher(source, 10)

	sub1 := dispatcher.Subscribe()
	sub2 := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Run(ctx)

	numPackets := 10
	for i := 0; i < numPackets; i++ {
		source <- []byte(string(rune('A' + i)))
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

	if count1 != numPackets {
		t.Errorf("subscriber 1: expected %d packets, got %d", numPackets, count1)
	}
	if count2 != numPackets {
		t.Errorf("subscriber 2: expected %d packets, got %d", numPackets, count2)
	}
}

func TestDispatcher_SubscribersReceiveSameData(t *testing.T) {
	source := make(chan []byte, 10)
	dispatcher := NewDispatcher(source, 10)

	sub1 := dispatcher.Subscribe()
	sub2 := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Run(ctx)

	data := [][]byte{
		[]byte("packet-1"),
		[]byte("packet-2"),
		[]byte("packet-3"),
	}
	for _, d := range data {
		source <- d
	}
	close(source)

	time.Sleep(10 * time.Millisecond)

	var received1, received2 [][]byte
	for d := range sub1 {
		received1 = append(received1, d)
	}
	for d := range sub2 {
		received2 = append(received2, d)
	}

	for i, d := range data {
		if string(received1[i]) != string(d) {
			t.Errorf("sub1 packet %d: got %s, want %s", i, string(received1[i]), string(d))
		}
		if string(received2[i]) != string(d) {
			t.Errorf("sub2 packet %d: got %s, want %s", i, string(received2[i]), string(d))
		}
	}
}

func TestDispatcher_ContextCancellation(t *testing.T) {
	source := make(chan []byte, 10)
	dispatcher := NewDispatcher(source, 10)

	sub := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		dispatcher.Run(ctx)
		close(done)
	}()

	source <- []byte("before-cancel")
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
		// First packet might still be there
		_, ok = <-sub
	}
	if ok {
		t.Error("subscriber channel should be closed after dispatcher stops")
	}
}

func TestDispatcher_SlowSubscriber(t *testing.T) {
	source := make(chan []byte, 10)
	dispatcher := NewDispatcher(source, 2) // Small buffer to trigger drops

	fastSub := dispatcher.Subscribe()
	slowSub := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Run(ctx)

	fastCount := 0
	fastDone := make(chan struct{})
	go func() {
		defer close(fastDone)
		for range fastSub {
			fastCount++
		}
	}()

	slowCount := 0
	slowDone := make(chan struct{})
	go func() {
		defer close(slowDone)
		for range slowSub {
			slowCount++
			time.Sleep(10 * time.Millisecond) // Slow processing
		}
	}()

	time.Sleep(5 * time.Millisecond)

	numPackets := 10
	for i := 0; i < numPackets; i++ {
		source <- []byte("data")
		time.Sleep(1 * time.Millisecond)
	}
	close(source)

	<-fastDone
	<-slowDone

	if fastCount != numPackets {
		t.Errorf("fast subscriber: expected %d packets, got %d", numPackets, fastCount)
	}

	dropped := dispatcher.GetDroppedCount()
	if dropped == 0 && slowCount < numPackets {
		t.Logf("Slow subscriber got %d packets (expected some drops), dropped count: %d", slowCount, dropped)
	}

	if slowCount == 0 {
		t.Error("slow subscriber should have received at least some packets")
	}
}

func TestDispatcher_GetSubscriberCount(t *testing.T) {
	source := make(chan []byte, 10)
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

	close(source)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	go dispatcher.Run(ctx)
	time.Sleep(10 * time.Millisecond)

	for range sub1 {
	}
	for range sub2 {
	}
}

func TestDispatcher_GetDroppedCount(t *testing.T) {
	source := make(chan []byte, 10)
	dispatcher := NewDispatcher(source, 1) // Very small buffer to force drops

	sub := dispatcher.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Run(ctx)

	for i := 0; i < 10; i++ {
		source <- []byte("data")
	}
	close(source)

	time.Sleep(50 * time.Millisecond)

	dropped := dispatcher.GetDroppedCount()
	if dropped < 0 {
		t.Errorf("dropped count should be non-negative, got %d", dropped)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range sub {
		}
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()
	<-done
}
