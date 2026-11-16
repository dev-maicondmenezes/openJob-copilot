package events

import (
	"testing"
	"time"
)

func TestBusFanOut(t *testing.T) {
	bus := New()
	a := bus.Subscribe("a", 8)
	defer a.Close()
	b := bus.Subscribe("b", 8)
	defer b.Close()

	bus.Publish(Event{Kind: "k1", Title: "t"})
	bus.Publish(Event{Kind: "k2", Title: "t"})

	for i, want := range []string{"k1", "k2"} {
		select {
		case e := <-a.C():
			if e.Kind != want {
				t.Errorf("a[%d].Kind = %s, want %s", i, e.Kind, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("a timed out waiting for %s", want)
		}
		select {
		case e := <-b.C():
			if e.Kind != want {
				t.Errorf("b[%d].Kind = %s, want %s", i, e.Kind, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("b timed out waiting for %s", want)
		}
	}
}

func TestBusPublishNeverBlocksOnFullSubscriber(t *testing.T) {
	bus := New()
	sub := bus.Subscribe("slow", 1) // buffer 1, nobody reads
	defer sub.Close()

	// Publish many events without blocking; the slow subscriber drops them.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			bus.Publish(Event{Kind: "spam"})
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a full subscriber")
	}
}

func TestBusUnsubscribeClosesChannel(t *testing.T) {
	bus := New()
	sub := bus.Subscribe("temp", 4)
	bus.Publish(Event{Kind: "k"})
	// Drain before closing to guarantee close() happens after all sends.
	<-sub.C()
	sub.Close()

	if _, ok := <-sub.C(); ok {
		t.Error("channel should be closed after Close")
	}
	// Publishing after unsubscribe must not panic.
	bus.Publish(Event{Kind: "late"})
}

func TestBusSubscribeSameNameReplaces(t *testing.T) {
	bus := New()
	first := bus.Subscribe("same", 4)
	second := bus.Subscribe("same", 4)
	defer second.Close()

	if _, ok := <-first.C(); ok {
		t.Error("replaced subscription channel should be closed")
	}
	bus.Publish(Event{Kind: "k"})
	select {
	case <-second.C():
	case <-time.After(time.Second):
		t.Fatal("replacement subscription did not receive the event")
	}
}
