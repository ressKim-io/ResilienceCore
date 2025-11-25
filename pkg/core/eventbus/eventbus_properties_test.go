package eventbus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
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
			bus := NewInMemoryEventBus()
			defer bus.Close()

			// Create multiple subscribers with matching filters
			subscriptions := make([]types.Subscription, numSubscribers)
			receivedCounts := make([]int, numSubscribers)
			var wg sync.WaitGroup

			for i := 0; i < numSubscribers; i++ {
				sub, err := bus.Subscribe(ctx, types.EventFilter{
					Types:   []string{eventType},
					Sources: []string{eventSource},
				})
				if err != nil {
					return false
				}
				subscriptions[i] = sub

				// Start goroutine to receive events
				wg.Add(1)
				go func(idx int, s types.Subscription) {
					defer wg.Done()
					timeout := time.After(50 * time.Millisecond)
					select {
					case <-s.Events():
						receivedCounts[idx]++
					case <-timeout:
						// Timeout is acceptable if no event was published yet
					}
				}(i, sub)
			}

			// Give goroutines time to start
			time.Sleep(5 * time.Millisecond)

			// Publish an event that matches all filters
			event := types.Event{
				ID:        "test-event-1",
				Type:      eventType,
				Source:    eventSource,
				Timestamp: time.Now(),
				Resource:  types.Resource{ID: "test-resource"},
				Data:      map[string]interface{}{"test": "data"},
				Metadata:  map[string]string{},
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
			bus := NewInMemoryEventBus()
			defer bus.Close()

			// Create subscription with specific filter
			sub, err := bus.Subscribe(ctx, types.EventFilter{
				Types:   []string{filterType},
				Sources: []string{filterSource},
			})
			if err != nil {
				return false
			}
			defer sub.Unsubscribe()

			// Publish event
			event := types.Event{
				ID:        "test-event",
				Type:      eventType,
				Source:    eventSource,
				Timestamp: time.Now(),
				Resource:  types.Resource{ID: "test-resource"},
				Metadata:  map[string]string{},
			}

			// Check if event should match
			shouldMatch := (eventType == filterType) && (eventSource == filterSource)

			// Publish event
			if err := bus.Publish(ctx, event); err != nil {
				return false
			}

			// Try to receive event with timeout
			received := false
			timeout := time.After(20 * time.Millisecond)
			select {
			case <-sub.Events():
				received = true
			case <-timeout:
				received = false
			}

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
			bus := NewInMemoryEventBus()
			defer bus.Close()

			// Create subscription
			sub, err := bus.Subscribe(ctx, types.EventFilter{
				Types:   []string{eventType},
				Sources: []string{eventSource},
			})
			if err != nil {
				return false
			}

			// Publish first event (should be received)
			event1 := types.Event{
				ID:        "test-event-1",
				Type:      eventType,
				Source:    eventSource,
				Timestamp: time.Now(),
				Resource:  types.Resource{ID: "test-resource"},
				Metadata:  map[string]string{},
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
			time.Sleep(5 * time.Millisecond)

			// Publish second event
			event2 := types.Event{
				ID:        "test-event-2",
				Type:      eventType,
				Source:    eventSource,
				Timestamp: time.Now(),
				Resource:  types.Resource{ID: "test-resource"},
				Metadata:  map[string]string{},
			}

			if err := bus.Publish(ctx, event2); err != nil {
				return false
			}

			// The old subscription channel should be closed
			// Reading from a closed channel returns immediately with zero value and ok=false
			_, oldSubOpen := <-sub.Events()

			// Verify: old subscription channel is closed
			return !oldSubOpen
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty41_ShutdownClosesAllChannelsGracefully verifies that
// calling Close closes all subscription channels without panics
// Feature: infrastructure-resilience-engine, Property 41: Shutdown closes all channels gracefully
// Validates: Requirements 8.4
func TestProperty41_ShutdownClosesAllChannelsGracefully(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("shutdown closes all channels gracefully", prop.ForAll(
		func(numSubscribers int) bool {
			// Limit number of subscribers to reasonable range
			if numSubscribers < 1 || numSubscribers > 10 {
				return true // Skip invalid inputs
			}

			ctx := context.Background()
			bus := NewInMemoryEventBus()

			// Create multiple subscribers
			subscriptions := make([]types.Subscription, numSubscribers)
			for i := 0; i < numSubscribers; i++ {
				sub, err := bus.Subscribe(ctx, types.EventFilter{})
				if err != nil {
					return false
				}
				subscriptions[i] = sub
			}

			// Close the bus
			if err := bus.Close(); err != nil {
				return false
			}

			// Verify all subscription channels are closed
			for _, sub := range subscriptions {
				select {
				case _, ok := <-sub.Events():
					if ok {
						return false // Channel should be closed
					}
				case <-time.After(50 * time.Millisecond):
					return false // Channel should be closed immediately
				}
			}

			return true
		},
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty42_EventFilteringSupportsTypeSourceAndMetadata verifies that
// EventFilter with Type, Source, or Metadata criteria only passes matching events
// Feature: infrastructure-resilience-engine, Property 42: Event filtering supports type, source, and metadata
// Validates: Requirements 8.5
func TestProperty42_EventFilteringSupportsTypeSourceAndMetadata(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("event filtering supports type, source, and metadata", prop.ForAll(
		func(filterType string, filterSource string, filterMetaKey string, filterMetaValue string,
			eventType string, eventSource string, eventMetaKey string, eventMetaValue string) bool {

			ctx := context.Background()
			bus := NewInMemoryEventBus()
			defer bus.Close()

			// Create subscription with all filter criteria
			sub, err := bus.Subscribe(ctx, types.EventFilter{
				Types:   []string{filterType},
				Sources: []string{filterSource},
				Metadata: map[string]string{
					filterMetaKey: filterMetaValue,
				},
			})
			if err != nil {
				return false
			}
			defer sub.Unsubscribe()

			// Publish event
			event := types.Event{
				ID:        "test-event",
				Type:      eventType,
				Source:    eventSource,
				Timestamp: time.Now(),
				Resource:  types.Resource{ID: "test-resource"},
				Metadata: map[string]string{
					eventMetaKey: eventMetaValue,
				},
			}

			// Check if event should match all criteria
			shouldMatch := (eventType == filterType) &&
				(eventSource == filterSource) &&
				(eventMetaKey == filterMetaKey && eventMetaValue == filterMetaValue)

			// Publish event
			if err := bus.Publish(ctx, event); err != nil {
				return false
			}

			// Try to receive event with timeout
			received := false
			timeout := time.After(20 * time.Millisecond)
			select {
			case <-sub.Events():
				received = true
			case <-timeout:
				received = false
			}

			// Verify: received if and only if all criteria match
			return received == shouldMatch
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
