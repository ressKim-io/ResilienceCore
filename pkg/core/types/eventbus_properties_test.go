// Package types provides property-based tests for the EventBus interface
package types

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty38_EventsAreDeliveredToAllMatchingSubscribers verifies that
// any published Event is delivered to all Subscriptions with matching EventFilters
// Feature: infrastructure-resilience-engine, Property 38: Events are delivered to all matching subscribers
// Validates: Requirements 8.1
func TestProperty38_EventsAreDeliveredToAllMatchingSubscribers(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("events are delivered to all matching subscribers", prop.ForAll(
		func(eventType string, eventSource string, numSubscribers int) bool {
			// Limit number of subscribers to reasonable range
			if numSubscribers < 1 || numSubscribers > 10 {
				return true // Skip invalid inputs
			}

			ctx := context.Background()
			bus := NewMockEventBus()

			// Create multiple subscribers with matching filters
			subscriptions := make([]Subscription, numSubscribers)
			receivedCounts := make([]int, numSubscribers)
			var wg sync.WaitGroup

			for i := 0; i < numSubscribers; i++ {
				sub, err := bus.Subscribe(ctx, EventFilter{
					Types:   []string{eventType},
					Sources: []string{eventSource},
				})
				if err != nil {
					return false
				}
				subscriptions[i] = sub

				// Start goroutine to receive events
				wg.Add(1)
				go func(idx int, s Subscription) {
					defer wg.Done()
					timeout := time.After(100 * time.Millisecond)
					select {
					case <-s.Events():
						receivedCounts[idx]++
					case <-timeout:
						// Timeout is acceptable if no event was published yet
					}
				}(i, sub)
			}

			// Publish an event that matches all filters
			event := Event{
				ID:        "test-event-1",
				Type:      eventType,
				Source:    eventSource,
				Timestamp: time.Now(),
				Resource:  Resource{ID: "test-resource"},
				Data:      map[string]interface{}{"test": "data"},
			}

			if err := bus.Publish(ctx, event); err != nil {
				return false
			}

			// Wait for all subscribers to receive (or timeout)
			wg.Wait()

			// Verify all subscribers received the event
			for i := 0; i < numSubscribers; i++ {
				if receivedCounts[i] != 1 {
					return false
				}
			}

			// Cleanup
			for _, sub := range subscriptions {
				sub.Unsubscribe()
			}
			bus.Close()

			return true
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty39_SubscriptionReceivesOnlyMatchingEvents verifies that
// any Subscription with an EventFilter only receives events matching the filter criteria
// Feature: infrastructure-resilience-engine, Property 39: Subscription receives only matching events
// Validates: Requirements 8.2
func TestProperty39_SubscriptionReceivesOnlyMatchingEvents(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("subscription receives only matching events", prop.ForAll(
		func(filterType string, filterSource string, eventType string, eventSource string) bool {
			ctx := context.Background()
			bus := NewMockEventBus()

			// Create subscription with specific filter
			sub, err := bus.Subscribe(ctx, EventFilter{
				Types:   []string{filterType},
				Sources: []string{filterSource},
			})
			if err != nil {
				return false
			}

			// Publish event
			event := Event{
				ID:        "test-event",
				Type:      eventType,
				Source:    eventSource,
				Timestamp: time.Now(),
				Resource:  Resource{ID: "test-resource"},
			}

			if err := bus.Publish(ctx, event); err != nil {
				return false
			}

			// Check if event should match
			shouldMatch := (eventType == filterType) && (eventSource == filterSource)

			// Try to receive event with timeout
			received := false
			timeout := time.After(50 * time.Millisecond)
			select {
			case <-sub.Events():
				received = true
			case <-timeout:
				received = false
			}

			// Cleanup
			sub.Unsubscribe()
			bus.Close()

			// Verify: received if and only if it should match
			return received == shouldMatch
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty40_UnsubscribeStopsEventDelivery verifies that
// calling Unsubscribe stops event delivery to that subscription
// Feature: infrastructure-resilience-engine, Property 40: Unsubscribe stops event delivery
// Validates: Requirements 8.3
func TestProperty40_UnsubscribeStopsEventDelivery(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("unsubscribe stops event delivery", prop.ForAll(
		func(eventType string, eventSource string) bool {
			ctx := context.Background()
			bus := NewMockEventBus()

			// Create subscription
			sub, err := bus.Subscribe(ctx, EventFilter{
				Types:   []string{eventType},
				Sources: []string{eventSource},
			})
			if err != nil {
				return false
			}

			// Publish first event (should be received)
			event1 := Event{
				ID:        "test-event-1",
				Type:      eventType,
				Source:    eventSource,
				Timestamp: time.Now(),
				Resource:  Resource{ID: "test-resource"},
			}

			if pubErr := bus.Publish(ctx, event1); pubErr != nil {
				return false
			}

			// Receive first event
			timeout1 := time.After(50 * time.Millisecond)
			receivedFirst := false
			select {
			case _, ok := <-sub.Events():
				if ok {
					receivedFirst = true
				}
			case <-timeout1:
				return false // Should have received first event
			}

			if !receivedFirst {
				return false
			}

			// Unsubscribe
			if unsubErr := sub.Unsubscribe(); unsubErr != nil {
				return false
			}

			// Give time for unsubscribe to complete
			time.Sleep(10 * time.Millisecond)

			// Create a new subscription to verify the bus still works
			sub2, err := bus.Subscribe(ctx, EventFilter{
				Types:   []string{eventType},
				Sources: []string{eventSource},
			})
			if err != nil {
				return false
			}

			// Publish second event
			event2 := Event{
				ID:        "test-event-2",
				Type:      eventType,
				Source:    eventSource,
				Timestamp: time.Now(),
				Resource:  Resource{ID: "test-resource"},
			}

			if err := bus.Publish(ctx, event2); err != nil {
				return false
			}

			// The new subscription should receive the event
			timeout2 := time.After(50 * time.Millisecond)
			newSubReceived := false
			select {
			case _, ok := <-sub2.Events():
				if ok {
					newSubReceived = true
				}
			case <-timeout2:
				return false // New subscription should have received
			}

			// The old subscription channel should be closed
			// Reading from a closed channel returns immediately with zero value and ok=false
			_, oldSubOpen := <-sub.Events()

			// Cleanup
			sub2.Unsubscribe()
			bus.Close()

			// Verify: new subscription received event, old subscription channel is closed
			return newSubReceived && !oldSubOpen
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// MockEventBus is a simple in-memory implementation of EventBus for testing
type MockEventBus struct {
	subscriptions map[string]*MockSubscription
	mu            sync.RWMutex
	closed        bool
}

// NewMockEventBus creates a new MockEventBus
func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		subscriptions: make(map[string]*MockSubscription),
	}
}

func (m *MockEventBus) Publish(ctx context.Context, event Event) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil
	}

	// Deliver to all matching subscriptions
	for _, sub := range m.subscriptions {
		if sub.matches(event) {
			select {
			case sub.events <- event:
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Millisecond):
				// Skip if subscriber is slow
			}
		}
	}

	return nil
}

func (m *MockEventBus) PublishAsync(ctx context.Context, event Event) error {
	go m.Publish(ctx, event)
	return nil
}

func (m *MockEventBus) Subscribe(ctx context.Context, filter EventFilter) (Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, nil
	}

	sub := &MockSubscription{
		id:     generateID(),
		filter: filter,
		events: make(chan Event, 10),
		bus:    m,
	}

	m.subscriptions[sub.id] = sub
	return sub, nil
}

func (m *MockEventBus) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true

	// Close all subscription channels
	for _, sub := range m.subscriptions {
		close(sub.events)
	}

	m.subscriptions = make(map[string]*MockSubscription)
	return nil
}

// MockSubscription is a mock implementation of Subscription
type MockSubscription struct {
	id     string
	filter EventFilter
	events chan Event
	bus    *MockEventBus
	closed bool
	mu     sync.Mutex
}

func (m *MockSubscription) ID() string {
	return m.id
}

func (m *MockSubscription) Events() <-chan Event {
	return m.events
}

func (m *MockSubscription) Unsubscribe() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true

	// Remove from bus
	m.bus.mu.Lock()
	delete(m.bus.subscriptions, m.id)
	m.bus.mu.Unlock()

	// Close channel
	close(m.events)

	return nil
}

func (m *MockSubscription) matches(event Event) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return false
	}

	// Check type filter
	if len(m.filter.Types) > 0 {
		typeMatch := false
		for _, t := range m.filter.Types {
			if t == event.Type {
				typeMatch = true
				break
			}
		}
		if !typeMatch {
			return false
		}
	}

	// Check source filter
	if len(m.filter.Sources) > 0 {
		sourceMatch := false
		for _, s := range m.filter.Sources {
			if s == event.Source {
				sourceMatch = true
				break
			}
		}
		if !sourceMatch {
			return false
		}
	}

	// Check metadata filter
	if len(m.filter.Metadata) > 0 {
		for key, value := range m.filter.Metadata {
			if event.Metadata[key] != value {
				return false
			}
		}
	}

	return true
}

// generateID generates a simple unique ID
var idCounter int
var idMutex sync.Mutex

func generateID() string {
	idMutex.Lock()
	defer idMutex.Unlock()
	idCounter++
	return string(rune('a' + (idCounter % 26)))
}
