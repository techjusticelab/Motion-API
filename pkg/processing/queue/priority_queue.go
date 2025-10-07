package queue

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// priorityQueue implements a thread-safe priority queue
type priorityQueue struct {
	name           string
	queueType      QueueType
	items          *priorityHeap
	mutex          sync.RWMutex
	cond           *sync.Cond
	maxSize        int
	closed         bool
	
	// Statistics
	completedItems int64
	failedItems    int64
	totalWaitTime  int64
	itemCount      int64
	
	// Configuration
	config *QueueConfig
}

// priorityHeap implements heap.Interface for QueueItem
type priorityHeap []*QueueItem

func (h priorityHeap) Len() int { return len(h) }

func (h priorityHeap) Less(i, j int) bool {
	// Higher priority items come first
	if h[i].Priority != h[j].Priority {
		return h[i].Priority > h[j].Priority
	}
	// For same priority, older items come first (FIFO)
	return h[i].CreatedAt.Before(h[j].CreatedAt)
}

func (h priorityHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *priorityHeap) Push(x interface{}) {
	*h = append(*h, x.(*QueueItem))
}

func (h *priorityHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue(config *QueueConfig) Queue {
	pq := &priorityQueue{
		name:      config.Name,
		queueType: config.Type,
		items:     &priorityHeap{},
		maxSize:   config.MaxSize,
		config:    config,
	}
	
	pq.cond = sync.NewCond(&pq.mutex)
	heap.Init(pq.items)
	
	return pq
}

// Enqueue adds an item to the queue
func (pq *priorityQueue) Enqueue(ctx context.Context, item *QueueItem) error {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	
	if pq.closed {
		return fmt.Errorf("queue %s is closed", pq.name)
	}
	
	// Check size limit
	if pq.maxSize > 0 && len(*pq.items) >= pq.maxSize {
		return fmt.Errorf("queue %s is full (max size: %d)", pq.name, pq.maxSize)
	}
	
	// Set default values
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	if item.MaxRetries == 0 {
		item.MaxRetries = pq.config.RetryAttempts
	}
	
	heap.Push(pq.items, item)
	atomic.AddInt64(&pq.itemCount, 1)
	
	// Notify waiting goroutines
	pq.cond.Signal()
	
	return nil
}

// Dequeue removes and returns the highest priority item
func (pq *priorityQueue) Dequeue(ctx context.Context) (*QueueItem, error) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	
	for len(*pq.items) == 0 && !pq.closed {
		// Wait for an item or context cancellation
		done := make(chan struct{})
		go func() {
			defer close(done)
			pq.cond.Wait()
		}()
		
		select {
		case <-ctx.Done():
			pq.cond.Broadcast() // Wake up the waiting goroutine
			<-done // Wait for it to finish
			return nil, ctx.Err()
		case <-done:
			// Continue the loop to check conditions again
		}
	}
	
	if pq.closed && len(*pq.items) == 0 {
		return nil, fmt.Errorf("queue %s is closed and empty", pq.name)
	}
	
	item := heap.Pop(pq.items).(*QueueItem)
	
	// Calculate wait time
	waitTime := time.Since(item.CreatedAt)
	atomic.AddInt64(&pq.totalWaitTime, int64(waitTime))
	
	return item, nil
}

// Peek returns the highest priority item without removing it
func (pq *priorityQueue) Peek(ctx context.Context) (*QueueItem, error) {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()
	
	if len(*pq.items) == 0 {
		return nil, fmt.Errorf("queue %s is empty", pq.name)
	}
	
	return (*pq.items)[0], nil
}

// Size returns the current number of items in the queue
func (pq *priorityQueue) Size() int {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()
	return len(*pq.items)
}

// IsEmpty returns true if the queue is empty
func (pq *priorityQueue) IsEmpty() bool {
	return pq.Size() == 0
}

// Clear removes all items from the queue
func (pq *priorityQueue) Clear() error {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	
	if pq.closed {
		return fmt.Errorf("queue %s is closed", pq.name)
	}
	
	*pq.items = (*pq.items)[:0]
	heap.Init(pq.items)
	
	return nil
}

// GetStats returns queue statistics
func (pq *priorityQueue) GetStats() *QueueStats {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()
	
	var avgWaitTime time.Duration
	itemCount := atomic.LoadInt64(&pq.itemCount)
	if itemCount > 0 {
		totalWait := atomic.LoadInt64(&pq.totalWaitTime)
		avgWaitTime = time.Duration(totalWait / itemCount)
	}
	
	completed := atomic.LoadInt64(&pq.completedItems)
	failed := atomic.LoadInt64(&pq.failedItems)
	
	return &QueueStats{
		Name:            pq.name,
		Type:            pq.queueType,
		Size:            len(*pq.items),
		PendingItems:    len(*pq.items),
		ProcessingItems: 0, // This would be tracked by worker pool
		CompletedItems:  completed,
		FailedItems:     failed,
		RetryItems:      pq.countRetryItems(),
		AverageWaitTime: avgWaitTime,
		LastUpdate:      time.Now(),
	}
}

// countRetryItems counts items waiting for retry
func (pq *priorityQueue) countRetryItems() int {
	count := 0
	now := time.Now()
	
	for _, item := range *pq.items {
		if item.NextRetry != nil && item.NextRetry.After(now) {
			count++
		}
	}
	
	return count
}

// Close closes the queue and releases resources
func (pq *priorityQueue) Close() error {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	
	if pq.closed {
		return nil
	}
	
	pq.closed = true
	pq.cond.Broadcast() // Wake up any waiting goroutines
	
	return nil
}

// RequeueForRetry re-queues an item for retry with backoff
func (pq *priorityQueue) RequeueForRetry(item *QueueItem) error {
	if item.RetryCount >= item.MaxRetries {
		atomic.AddInt64(&pq.failedItems, 1)
		return fmt.Errorf("item %s exceeded max retries (%d)", item.ID, item.MaxRetries)
	}
	
	item.RetryCount++
	
	// Calculate retry delay with exponential backoff
	delay := pq.config.RetryDelay
	if pq.config.RetryBackoff && item.RetryCount > 1 {
		delay = delay * time.Duration(1<<uint(item.RetryCount-1))
		// Cap at 5 minutes
		if delay > 5*time.Minute {
			delay = 5 * time.Minute
		}
	}
	
	nextRetry := time.Now().Add(delay)
	item.NextRetry = &nextRetry
	
	// Re-queue the item
	return pq.Enqueue(context.Background(), item)
}

// MarkCompleted marks an item as completed
func (pq *priorityQueue) MarkCompleted(item *QueueItem) {
	now := time.Now()
	item.ProcessedAt = &now
	atomic.AddInt64(&pq.completedItems, 1)
}

// MarkFailed marks an item as failed
func (pq *priorityQueue) MarkFailed(item *QueueItem) {
	now := time.Now()
	item.ProcessedAt = &now
	atomic.AddInt64(&pq.failedItems, 1)
}

// GetReadyItems returns items that are ready for processing (not waiting for retry)
func (pq *priorityQueue) GetReadyItems(ctx context.Context) ([]*QueueItem, error) {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()
	
	var readyItems []*QueueItem
	now := time.Now()
	
	for _, item := range *pq.items {
		if item.NextRetry == nil || item.NextRetry.Before(now) {
			readyItems = append(readyItems, item)
		}
	}
	
	return readyItems, nil
}

// RemoveItem removes a specific item from the queue
func (pq *priorityQueue) RemoveItem(itemID string) bool {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	
	for i, item := range *pq.items {
		if item.ID == itemID {
			// Remove item at index i
			(*pq.items)[i] = (*pq.items)[len(*pq.items)-1]
			*pq.items = (*pq.items)[:len(*pq.items)-1]
			heap.Init(pq.items) // Re-heapify
			return true
		}
	}
	
	return false
}

// UpdatePriority updates the priority of an item in the queue
func (pq *priorityQueue) UpdatePriority(itemID string, newPriority Priority) error {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	
	for _, item := range *pq.items {
		if item.ID == itemID {
			item.Priority = newPriority
			heap.Init(pq.items) // Re-heapify after priority change
			return nil
		}
	}
	
	return fmt.Errorf("item %s not found in queue %s", itemID, pq.name)
}

// GetItemsByPriority returns items grouped by priority
func (pq *priorityQueue) GetItemsByPriority() map[Priority][]*QueueItem {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()
	
	priorityGroups := make(map[Priority][]*QueueItem)
	
	for _, item := range *pq.items {
		priorityGroups[item.Priority] = append(priorityGroups[item.Priority], item)
	}
	
	return priorityGroups
}