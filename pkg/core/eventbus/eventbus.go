// Package eventbus provides an in-memory implementation of the EventBus interface
// for loose coupling between components through publish-subscribe messaging.
package eventbus

import (
	"context"
	"fmt"
	"sync"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// InMemoryEventBus is an in-memory implementation of the EventBus interface
type InMemoryEventBus struct {
	mu            sync.RWMutex
	subscriptions map[string]*subscription
	nextID        int
	closed        bool
}

// subscription represents an active subscription
type subscription struct {
	id       string
	filter   types.EventFilter
	events   chan types.Event
	unsubbed bool
	mu       sync.Mutex
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		subscriptions: make(map[string]*subscription),
		nextID:        1,
	}
}

// Publish publishes an event synchronously to all matching subscribers
func (bus *InMemoryEventBus) Publish(ctx context.Context, event types.Event) error {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	if bus.closed {
		return fmt.Errorf("event bus is closed")
	}

	// Deliver to all matching subscriptions
	for _, sub := range bus.subscriptions {
		if bus.matchesFilter(event, sub.filter) {
			sub.mu.Lock()
			if !sub.unsubbed {
				select {
				case sub.events <- event:
				case <-ctx.Done():
					sub.mu.Unlock()
					return ctx.Err()
				default:
					// Non-blocking send - if channel is full, skip this subscriber
				}
			}
			sub.mu.Unlock()
		}
	}

	return nil
}

// PublishAsync publishes an event asynchronously to all matching subscribers
func (bus *InMemoryEventBus) PublishAsync(ctx context.Context, event types.Event) error {
	go func() {
		// Intentionally ignore error in async publish
		// Errors are logged but don't block the caller
		if err := bus.Publish(ctx, event); err != nil {
			// In a production system, this would be logged
			// For now, we silently ignore errors in async mode
			_ = err
		}
	}()
	return nil
}

// Subscribe creates a new subscription with the given filter
func (bus *InMemoryEventBus) Subscribe(ctx context.Context, filter types.EventFilter) (types.Subscription, error) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.closed {
		return nil, fmt.Errorf("event bus is closed")
	}

	// Create new subscription
	sub := &subscription{
		id:     fmt.Sprintf("sub-%d", bus.nextID),
		filter: filter,
		events: make(chan types.Event, 100), // Buffered channel
	}
	bus.nextID++

	bus.subscriptions[sub.id] = sub

	return &subscriptionImpl{
		sub: sub,
		bus: bus,
	}, nil
}

// Close closes the event bus and all subscription channels
func (bus *InMemoryEventBus) Close() error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.closed {
		return nil
	}

	bus.closed = true

	// Close all subscription channels
	for _, sub := range bus.subscriptions {
		sub.mu.Lock()
		if !sub.unsubbed {
			close(sub.events)
			sub.unsubbed = true
		}
		sub.mu.Unlock()
	}

	// Clear subscriptions
	bus.subscriptions = make(map[string]*subscription)

	return nil
}

// matchesFilter checks if an event matches the given filter
func (bus *InMemoryEventBus) matchesFilter(event types.Event, filter types.EventFilter) bool {
	// Check type filter
	if len(filter.Types) > 0 {
		matched := false
		for _, t := range filter.Types {
			if event.Type == t {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check source filter
	if len(filter.Sources) > 0 {
		matched := false
		for _, s := range filter.Sources {
			if event.Source == s {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check metadata filter
	if len(filter.Metadata) > 0 {
		for key, value := range filter.Metadata {
			if eventValue, ok := event.Metadata[key]; !ok || eventValue != value {
				return false
			}
		}
	}

	return true
}

// subscriptionImpl implements the Subscription interface
type subscriptionImpl struct {
	sub *subscription
	bus *InMemoryEventBus
}

// ID returns the subscription ID
func (s *subscriptionImpl) ID() string {
	return s.sub.id
}

// Events returns the channel for receiving events
func (s *subscriptionImpl) Events() <-chan types.Event {
	return s.sub.events
}

// Unsubscribe removes the subscription and closes its channel
func (s *subscriptionImpl) Unsubscribe() error {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()

	s.sub.mu.Lock()
	defer s.sub.mu.Unlock()

	if s.sub.unsubbed {
		return nil
	}

	// Mark as unsubscribed
	s.sub.unsubbed = true

	// Close the channel
	close(s.sub.events)

	// Remove from bus
	delete(s.bus.subscriptions, s.sub.id)

	return nil
}
