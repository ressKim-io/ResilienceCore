package eventbus

import (
	"context"
	"testing"
	"time"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

func TestInMemoryEventBus_PublishAndSubscribe(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	ctx := context.Background()

	// Subscribe to events
	sub, err := bus.Subscribe(ctx, types.EventFilter{})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	// Publish an event
	event := types.Event{
		ID:        "test-1",
		Type:      "test.event",
		Source:    "test",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"key": "value"},
		Metadata:  map[string]string{"meta": "data"},
	}

	err = bus.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Receive the event
	select {
	case received := <-sub.Events():
		if received.ID != event.ID {
			t.Errorf("Expected event ID %s, got %s", event.ID, received.ID)
		}
		if received.Type != event.Type {
			t.Errorf("Expected event type %s, got %s", event.Type, received.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestInMemoryEventBus_PublishAsync(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	ctx := context.Background()

	// Subscribe to events
	sub, err := bus.Subscribe(ctx, types.EventFilter{})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	// Publish an event asynchronously
	event := types.Event{
		ID:        "test-async-1",
		Type:      "test.async",
		Source:    "test",
		Timestamp: time.Now(),
	}

	err = bus.PublishAsync(ctx, event)
	if err != nil {
		t.Fatalf("PublishAsync failed: %v", err)
	}

	// Receive the event
	select {
	case received := <-sub.Events():
		if received.ID != event.ID {
			t.Errorf("Expected event ID %s, got %s", event.ID, received.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for async event")
	}
}

func TestInMemoryEventBus_FilterByType(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	ctx := context.Background()

	// Subscribe with type filter
	sub, err := bus.Subscribe(ctx, types.EventFilter{
		Types: []string{"test.filtered"},
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	// Publish matching event
	matchingEvent := types.Event{
		ID:        "matching",
		Type:      "test.filtered",
		Source:    "test",
		Timestamp: time.Now(),
	}
	bus.Publish(ctx, matchingEvent)

	// Publish non-matching event
	nonMatchingEvent := types.Event{
		ID:        "non-matching",
		Type:      "test.other",
		Source:    "test",
		Timestamp: time.Now(),
	}
	bus.Publish(ctx, nonMatchingEvent)

	// Should only receive matching event
	select {
	case received := <-sub.Events():
		if received.ID != matchingEvent.ID {
			t.Errorf("Expected matching event, got %s", received.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for matching event")
	}

	// Should not receive non-matching event
	select {
	case received := <-sub.Events():
		t.Errorf("Received unexpected event: %s", received.ID)
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}
}

func TestInMemoryEventBus_FilterBySource(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	ctx := context.Background()

	// Subscribe with source filter
	sub, err := bus.Subscribe(ctx, types.EventFilter{
		Sources: []string{"source-a"},
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	// Publish matching event
	matchingEvent := types.Event{
		ID:        "matching",
		Type:      "test.event",
		Source:    "source-a",
		Timestamp: time.Now(),
	}
	bus.Publish(ctx, matchingEvent)

	// Publish non-matching event
	nonMatchingEvent := types.Event{
		ID:        "non-matching",
		Type:      "test.event",
		Source:    "source-b",
		Timestamp: time.Now(),
	}
	bus.Publish(ctx, nonMatchingEvent)

	// Should only receive matching event
	select {
	case received := <-sub.Events():
		if received.ID != matchingEvent.ID {
			t.Errorf("Expected matching event, got %s", received.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for matching event")
	}

	// Should not receive non-matching event
	select {
	case received := <-sub.Events():
		t.Errorf("Received unexpected event: %s", received.ID)
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}
}

func TestInMemoryEventBus_FilterByMetadata(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	ctx := context.Background()

	// Subscribe with metadata filter
	sub, err := bus.Subscribe(ctx, types.EventFilter{
		Metadata: map[string]string{
			"env": "production",
		},
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	// Publish matching event
	matchingEvent := types.Event{
		ID:        "matching",
		Type:      "test.event",
		Source:    "test",
		Timestamp: time.Now(),
		Metadata:  map[string]string{"env": "production"},
	}
	bus.Publish(ctx, matchingEvent)

	// Publish non-matching event
	nonMatchingEvent := types.Event{
		ID:        "non-matching",
		Type:      "test.event",
		Source:    "test",
		Timestamp: time.Now(),
		Metadata:  map[string]string{"env": "development"},
	}
	bus.Publish(ctx, nonMatchingEvent)

	// Should only receive matching event
	select {
	case received := <-sub.Events():
		if received.ID != matchingEvent.ID {
			t.Errorf("Expected matching event, got %s", received.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for matching event")
	}

	// Should not receive non-matching event
	select {
	case received := <-sub.Events():
		t.Errorf("Received unexpected event: %s", received.ID)
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}
}

func TestInMemoryEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	ctx := context.Background()

	// Create multiple subscribers
	sub1, err := bus.Subscribe(ctx, types.EventFilter{})
	if err != nil {
		t.Fatalf("Subscribe 1 failed: %v", err)
	}
	defer sub1.Unsubscribe()

	sub2, err := bus.Subscribe(ctx, types.EventFilter{})
	if err != nil {
		t.Fatalf("Subscribe 2 failed: %v", err)
	}
	defer sub2.Unsubscribe()

	// Publish an event
	event := types.Event{
		ID:        "multi-sub",
		Type:      "test.event",
		Source:    "test",
		Timestamp: time.Now(),
	}
	bus.Publish(ctx, event)

	// Both subscribers should receive the event
	select {
	case received := <-sub1.Events():
		if received.ID != event.ID {
			t.Errorf("Sub1: Expected event ID %s, got %s", event.ID, received.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Sub1: Timeout waiting for event")
	}

	select {
	case received := <-sub2.Events():
		if received.ID != event.ID {
			t.Errorf("Sub2: Expected event ID %s, got %s", event.ID, received.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Sub2: Timeout waiting for event")
	}
}

func TestInMemoryEventBus_Unsubscribe(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	ctx := context.Background()

	// Subscribe
	sub, err := bus.Subscribe(ctx, types.EventFilter{})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Publish first event
	event1 := types.Event{
		ID:        "before-unsub",
		Type:      "test.event",
		Source:    "test",
		Timestamp: time.Now(),
	}
	bus.Publish(ctx, event1)

	// Receive first event
	select {
	case received := <-sub.Events():
		if received.ID != event1.ID {
			t.Errorf("Expected event ID %s, got %s", event1.ID, received.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for first event")
	}

	// Unsubscribe
	err = sub.Unsubscribe()
	if err != nil {
		t.Fatalf("Unsubscribe failed: %v", err)
	}

	// Publish second event
	event2 := types.Event{
		ID:        "after-unsub",
		Type:      "test.event",
		Source:    "test",
		Timestamp: time.Now(),
	}
	bus.Publish(ctx, event2)

	// Should not receive second event (channel should be closed)
	select {
	case _, ok := <-sub.Events():
		if ok {
			t.Error("Received event after unsubscribe")
		}
		// Channel closed as expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Channel not closed after unsubscribe")
	}
}

func TestInMemoryEventBus_Close(t *testing.T) {
	bus := NewInMemoryEventBus()
	ctx := context.Background()

	// Create subscriptions
	sub1, err := bus.Subscribe(ctx, types.EventFilter{})
	if err != nil {
		t.Fatalf("Subscribe 1 failed: %v", err)
	}

	sub2, err := bus.Subscribe(ctx, types.EventFilter{})
	if err != nil {
		t.Fatalf("Subscribe 2 failed: %v", err)
	}

	// Close the bus
	err = bus.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// All subscription channels should be closed
	select {
	case _, ok := <-sub1.Events():
		if ok {
			t.Error("Sub1 channel not closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Sub1 channel not closed after bus close")
	}

	select {
	case _, ok := <-sub2.Events():
		if ok {
			t.Error("Sub2 channel not closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Sub2 channel not closed after bus close")
	}

	// Publishing after close should fail
	event := types.Event{
		ID:        "after-close",
		Type:      "test.event",
		Source:    "test",
		Timestamp: time.Now(),
	}
	err = bus.Publish(ctx, event)
	if err == nil {
		t.Error("Expected error when publishing to closed bus")
	}

	// Subscribing after close should fail
	_, err = bus.Subscribe(ctx, types.EventFilter{})
	if err == nil {
		t.Error("Expected error when subscribing to closed bus")
	}
}

func TestInMemoryEventBus_CombinedFilters(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	ctx := context.Background()

	// Subscribe with combined filters
	sub, err := bus.Subscribe(ctx, types.EventFilter{
		Types:   []string{"test.event"},
		Sources: []string{"source-a"},
		Metadata: map[string]string{
			"env": "production",
		},
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	// Publish fully matching event
	matchingEvent := types.Event{
		ID:        "matching",
		Type:      "test.event",
		Source:    "source-a",
		Timestamp: time.Now(),
		Metadata:  map[string]string{"env": "production"},
	}
	bus.Publish(ctx, matchingEvent)

	// Publish partially matching events
	wrongType := types.Event{
		ID:        "wrong-type",
		Type:      "other.event",
		Source:    "source-a",
		Timestamp: time.Now(),
		Metadata:  map[string]string{"env": "production"},
	}
	bus.Publish(ctx, wrongType)

	wrongSource := types.Event{
		ID:        "wrong-source",
		Type:      "test.event",
		Source:    "source-b",
		Timestamp: time.Now(),
		Metadata:  map[string]string{"env": "production"},
	}
	bus.Publish(ctx, wrongSource)

	wrongMetadata := types.Event{
		ID:        "wrong-metadata",
		Type:      "test.event",
		Source:    "source-a",
		Timestamp: time.Now(),
		Metadata:  map[string]string{"env": "development"},
	}
	bus.Publish(ctx, wrongMetadata)

	// Should only receive fully matching event
	select {
	case received := <-sub.Events():
		if received.ID != matchingEvent.ID {
			t.Errorf("Expected matching event, got %s", received.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for matching event")
	}

	// Should not receive any other events
	select {
	case received := <-sub.Events():
		t.Errorf("Received unexpected event: %s", received.ID)
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}
}
