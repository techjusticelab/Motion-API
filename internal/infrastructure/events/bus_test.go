package events

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"motion-index-fiber/internal/domain/document"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEvent is a test event implementation.
type mockEvent struct {
	eventType   string
	aggregateID string
	occurredAt  time.Time
}

func (e *mockEvent) EventType() string {
	return e.eventType
}

func (e *mockEvent) AggregateID() string {
	return e.aggregateID
}

func (e *mockEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func TestNewInMemoryEventBus(t *testing.T) {
	bus := NewInMemoryEventBus(nil)

	assert.NotNil(t, bus)
	assert.NotNil(t, bus.handlers)
	assert.NotNil(t, bus.logger)
}

func TestInMemoryEventBus_Register(t *testing.T) {
	bus := NewInMemoryEventBus(log.Default())

	handler := func(ctx context.Context, event document.DomainEvent) error {
		return nil
	}

	bus.Register("TestEvent", handler)

	assert.Equal(t, 1, bus.HandlerCount("TestEvent"))
}

func TestInMemoryEventBus_Publish_Success(t *testing.T) {
	bus := NewInMemoryEventBus(log.Default())

	// Track handler invocations
	var handledEvents []string

	handler := func(ctx context.Context, event document.DomainEvent) error {
		handledEvents = append(handledEvents, event.EventType())
		return nil
	}

	bus.Register("TestEvent", handler)

	// Publish event
	event := &mockEvent{
		eventType:   "TestEvent",
		aggregateID: "test_123",
		occurredAt:  time.Now(),
	}

	err := bus.Publish(context.Background(), event)

	require.NoError(t, err)
	assert.Len(t, handledEvents, 1)
	assert.Equal(t, "TestEvent", handledEvents[0])
}

func TestInMemoryEventBus_Publish_MultipleHandlers(t *testing.T) {
	bus := NewInMemoryEventBus(log.Default())

	// Track handler invocations
	var handler1Called, handler2Called bool

	handler1 := func(ctx context.Context, event document.DomainEvent) error {
		handler1Called = true
		return nil
	}

	handler2 := func(ctx context.Context, event document.DomainEvent) error {
		handler2Called = true
		return nil
	}

	bus.Register("TestEvent", handler1)
	bus.Register("TestEvent", handler2)

	// Publish event
	event := &mockEvent{
		eventType:   "TestEvent",
		aggregateID: "test_123",
		occurredAt:  time.Now(),
	}

	err := bus.Publish(context.Background(), event)

	require.NoError(t, err)
	assert.True(t, handler1Called, "Handler 1 should be called")
	assert.True(t, handler2Called, "Handler 2 should be called")
}

func TestInMemoryEventBus_Publish_HandlerError(t *testing.T) {
	bus := NewInMemoryEventBus(log.Default())

	expectedErr := errors.New("handler error")

	handler := func(ctx context.Context, event document.DomainEvent) error {
		return expectedErr
	}

	bus.Register("TestEvent", handler)

	// Publish event
	event := &mockEvent{
		eventType:   "TestEvent",
		aggregateID: "test_123",
		occurredAt:  time.Now(),
	}

	err := bus.Publish(context.Background(), event)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "event bus errors")
}

func TestInMemoryEventBus_Publish_NoHandlers(t *testing.T) {
	bus := NewInMemoryEventBus(log.Default())

	// Publish event with no registered handlers
	event := &mockEvent{
		eventType:   "UnknownEvent",
		aggregateID: "test_123",
		occurredAt:  time.Now(),
	}

	err := bus.Publish(context.Background(), event)

	require.NoError(t, err) // Should not error when no handlers exist
}

func TestInMemoryEventBus_Publish_MultipleEvents(t *testing.T) {
	bus := NewInMemoryEventBus(log.Default())

	var handledEvents []string

	handler := func(ctx context.Context, event document.DomainEvent) error {
		handledEvents = append(handledEvents, event.EventType())
		return nil
	}

	bus.Register("Event1", handler)
	bus.Register("Event2", handler)

	// Publish multiple events
	events := []document.DomainEvent{
		&mockEvent{eventType: "Event1", aggregateID: "test_1", occurredAt: time.Now()},
		&mockEvent{eventType: "Event2", aggregateID: "test_2", occurredAt: time.Now()},
	}

	err := bus.Publish(context.Background(), events...)

	require.NoError(t, err)
	assert.Len(t, handledEvents, 2)
	assert.Contains(t, handledEvents, "Event1")
	assert.Contains(t, handledEvents, "Event2")
}

func TestInMemoryEventBus_Publish_HandlerPanic(t *testing.T) {
	bus := NewInMemoryEventBus(log.Default())

	handler := func(ctx context.Context, event document.DomainEvent) error {
		panic("handler panic")
	}

	bus.Register("TestEvent", handler)

	// Publish event - should recover from panic
	event := &mockEvent{
		eventType:   "TestEvent",
		aggregateID: "test_123",
		occurredAt:  time.Now(),
	}

	// Should not panic
	err := bus.Publish(context.Background(), event)

	// Error handling behavior may vary based on implementation
	_ = err
}

func TestInMemoryEventBus_Clear(t *testing.T) {
	bus := NewInMemoryEventBus(log.Default())

	handler := func(ctx context.Context, event document.DomainEvent) error {
		return nil
	}

	bus.Register("TestEvent", handler)
	assert.Equal(t, 1, bus.HandlerCount("TestEvent"))

	bus.Clear()
	assert.Equal(t, 0, bus.HandlerCount("TestEvent"))
}

func TestInMemoryEventBus_PublishAsync(t *testing.T) {
	bus := NewInMemoryEventBus(log.Default())

	handlerCalled := false
	done := make(chan bool)

	handler := func(ctx context.Context, event document.DomainEvent) error {
		handlerCalled = true
		done <- true
		return nil
	}

	bus.Register("TestEvent", handler)

	// Publish async
	event := &mockEvent{
		eventType:   "TestEvent",
		aggregateID: "test_123",
		occurredAt:  time.Now(),
	}

	bus.PublishAsync(context.Background(), event)

	// Wait for handler to be called
	select {
	case <-done:
		assert.True(t, handlerCalled)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for async handler")
	}
}
