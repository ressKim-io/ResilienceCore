# EventBus Package

This package provides an in-memory implementation of the EventBus interface for the Infrastructure Resilience Engine.

## Overview

The EventBus enables loose coupling between components through publish-subscribe messaging. Components can publish events and subscribe to events matching specific filters without direct dependencies on each other.

## Implementation

### InMemoryEventBus

The `InMemoryEventBus` is a thread-safe, in-memory implementation that:

- Supports synchronous and asynchronous event publishing
- Provides flexible event filtering by type, source, and metadata
- Delivers events to all matching subscribers
- Handles subscription lifecycle (subscribe/unsubscribe)
- Gracefully closes all channels on shutdown

### Key Features

1. **Thread-Safe**: Uses RWMutex for concurrent access to subscriptions
2. **Buffered Channels**: Each subscription has a buffered channel (100 events) to prevent blocking
3. **Non-Blocking Publish**: If a subscriber's channel is full, the event is skipped for that subscriber
4. **Graceful Shutdown**: Closes all subscription channels without panics
5. **Flexible Filtering**: Supports filtering by event type, source, and metadata with AND semantics

## Usage

```go
// Create a new event bus
bus := eventbus.NewInMemoryEventBus()
defer bus.Close()

// Subscribe to events
ctx := context.Background()
sub, err := bus.Subscribe(ctx, types.EventFilter{
    Types:   []string{"resource.created"},
    Sources: []string{"adapter"},
})
if err != nil {
    log.Fatal(err)
}
defer sub.Unsubscribe()

// Publish an event
event := types.Event{
    ID:        "evt-123",
    Type:      "resource.created",
    Source:    "adapter",
    Timestamp: time.Now(),
    Resource:  resource,
    Data:      map[string]interface{}{"action": "create"},
    Metadata:  map[string]string{"env": "production"},
}

if err := bus.Publish(ctx, event); err != nil {
    log.Fatal(err)
}

// Receive events
for event := range sub.Events() {
    fmt.Printf("Received event: %s\n", event.ID)
}
```

## Event Filtering

The EventBus supports three types of filters:

1. **Type Filter**: Match events by type (e.g., "resource.created", "plugin.executed")
2. **Source Filter**: Match events by source (e.g., "adapter", "engine", "monitor")
3. **Metadata Filter**: Match events by metadata key-value pairs

All filter criteria use AND semantics - an event must match all specified criteria to be delivered.

### Example Filters

```go
// Match only resource.created events
filter := types.EventFilter{
    Types: []string{"resource.created"},
}

// Match events from specific source
filter := types.EventFilter{
    Sources: []string{"k8s-adapter"},
}

// Match events with specific metadata
filter := types.EventFilter{
    Metadata: map[string]string{
        "env": "production",
        "region": "us-west-2",
    },
}

// Combined filter (all criteria must match)
filter := types.EventFilter{
    Types:   []string{"resource.created", "resource.updated"},
    Sources: []string{"k8s-adapter"},
    Metadata: map[string]string{
        "env": "production",
    },
}
```

## Testing

The package includes comprehensive tests:

### Unit Tests

- Basic publish/subscribe functionality
- Filtering by type, source, and metadata
- Multiple subscribers
- Unsubscribe behavior
- Close/shutdown behavior
- Combined filters

### Property-Based Tests

The implementation is verified against the following correctness properties:

1. **Property 38**: Events are delivered to all matching subscribers
2. **Property 39**: Subscription receives only matching events
3. **Property 40**: Unsubscribe stops event delivery
4. **Property 41**: Shutdown closes all channels gracefully
5. **Property 42**: Event filtering supports type, source, and metadata

All properties are tested with 100+ iterations using gopter.

## Thread Safety

The InMemoryEventBus is designed for concurrent use:

- Multiple goroutines can publish events simultaneously
- Multiple goroutines can subscribe/unsubscribe simultaneously
- Subscription channels are buffered to prevent blocking publishers
- Proper synchronization ensures no race conditions

## Performance Considerations

- **Buffered Channels**: Each subscription has a 100-event buffer
- **Non-Blocking Publish**: Slow subscribers don't block publishers
- **Efficient Filtering**: Filter matching is done in-memory with simple comparisons
- **Lock Granularity**: Uses RWMutex for read-heavy workloads

## Limitations

- **In-Memory Only**: Events are not persisted
- **Single Process**: Does not support distributed event bus
- **No Ordering Guarantees**: Events may be delivered in different orders to different subscribers
- **Buffer Overflow**: If a subscriber's buffer is full, events are dropped for that subscriber

## Future Enhancements

Potential improvements for future versions:

- Persistent event storage
- Distributed event bus (Redis, NATS, Kafka)
- Event replay capabilities
- Delivery guarantees (at-least-once, exactly-once)
- Event ordering guarantees
- Backpressure handling
- Metrics and observability
