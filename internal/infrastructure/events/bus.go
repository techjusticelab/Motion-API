package events

import (
	"context"
	"fmt"
	"log"
	"sync"

	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/document"
)

// EventHandler is a function that handles a domain event.
type EventHandler func(ctx context.Context, event document.DomainEvent) error

// InMemoryEventBus is a simple synchronous in-memory event bus implementation.
type InMemoryEventBus struct {
	handlers map[string][]EventHandler // eventType -> handlers
	mu       sync.RWMutex
	logger   *log.Logger
}

// NewInMemoryEventBus creates a new in-memory event bus.
func NewInMemoryEventBus(logger *log.Logger) *InMemoryEventBus {
	if logger == nil {
		logger = log.Default()
	}

	return &InMemoryEventBus{
		handlers: make(map[string][]EventHandler),
		logger:   logger,
	}
}

// Register registers an event handler for a specific event type.
func (b *InMemoryEventBus) Register(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.logger.Printf("[EventBus] Registered handler for event type: %s", eventType)
}

// Publish implements the EventBus port.
// It dispatches events synchronously to all registered handlers.
func (b *InMemoryEventBus) Publish(ctx context.Context, events ...document.DomainEvent) error {
	if len(events) == 0 {
		return nil
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	var errors []error

	for _, event := range events {
		eventType := event.EventType()
		handlers, exists := b.handlers[eventType]

		if !exists || len(handlers) == 0 {
			b.logger.Printf("[EventBus] No handlers registered for event type: %s", eventType)
			continue
		}

		b.logger.Printf("[EventBus] Publishing event: %s (aggregate: %s)",
			eventType, event.AggregateID())

		// Dispatch to all registered handlers
		for i, handler := range handlers {
			if err := b.dispatchToHandler(ctx, handler, event, i); err != nil {
				errors = append(errors, err)
			}
		}
	}

	// Return combined errors if any handlers failed
	if len(errors) > 0 {
		return fmt.Errorf("event bus errors: %v", errors)
	}

	return nil
}

// dispatchToHandler dispatches an event to a single handler with error recovery.
func (b *InMemoryEventBus) dispatchToHandler(
	ctx context.Context,
	handler EventHandler,
	event document.DomainEvent,
	handlerIndex int,
) error {
	// Recover from panics in handlers
	defer func() {
		if r := recover(); r != nil {
			b.logger.Printf("[EventBus] Handler %d panicked for event %s: %v",
				handlerIndex, event.EventType(), r)
		}
	}()

	// Execute handler
	if err := handler(ctx, event); err != nil {
		b.logger.Printf("[EventBus] Handler %d failed for event %s: %v",
			handlerIndex, event.EventType(), err)
		return fmt.Errorf("handler %d failed: %w", handlerIndex, err)
	}

	return nil
}

// PublishAsync publishes events asynchronously (fire and forget).
// This is useful when you don't want to block on event processing.
func (b *InMemoryEventBus) PublishAsync(ctx context.Context, events ...document.DomainEvent) {
	go func() {
		if err := b.Publish(ctx, events...); err != nil {
			b.logger.Printf("[EventBus] Async publish failed: %v", err)
		}
	}()
}

// HandlerCount returns the number of handlers registered for an event type.
func (b *InMemoryEventBus) HandlerCount(eventType string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.handlers[eventType])
}

// Clear removes all registered handlers. Useful for testing.
func (b *InMemoryEventBus) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers = make(map[string][]EventHandler)
	b.logger.Printf("[EventBus] Cleared all handlers")
}

// Ensure InMemoryEventBus implements the EventBus port
var _ ports.EventBus = (*InMemoryEventBus)(nil)
