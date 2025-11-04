package opensearch

import (
	"context"
	"errors"
	"sync"
)

type dryJobPayload struct {
	FileName    string
	ContentType string
	Size        int64
	TempPath    string
}

type dryJobResult struct {
	status int
	body   interface{}
	err    error
}

type dryJob struct {
	ctx     context.Context
	payload *dryJobPayload
	result  chan dryJobResult
}

type dryJobQueue struct {
	mu        sync.Mutex
	cond      *sync.Cond
	jobs      []*dryJob
	processor func(*dryJob) dryJobResult
	closed    bool
}

func newDryJobQueue(processor func(*dryJob) dryJobResult) *dryJobQueue {
	q := &dryJobQueue{
		processor: processor,
		jobs:      make([]*dryJob, 0),
	}
	q.cond = sync.NewCond(&q.mu)
	go q.run()
	return q
}

func (q *dryJobQueue) run() {
	for {
		q.mu.Lock()
		for len(q.jobs) == 0 && !q.closed {
			q.cond.Wait()
		}
		if q.closed {
			q.mu.Unlock()
			return
		}
		job := q.jobs[0]
		q.jobs = q.jobs[1:]
		q.mu.Unlock()

		result := q.processor(job)
		job.result <- result
	}
}

func (q *dryJobQueue) Submit(ctx context.Context, payload *dryJobPayload) (dryJobResult, error) {
	if payload == nil {
		return dryJobResult{}, errors.New("nil payload")
	}

	job := &dryJob{
		ctx:     ctx,
		payload: payload,
		result:  make(chan dryJobResult, 1),
	}

	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return dryJobResult{}, errors.New("job queue closed")
	}
	q.jobs = append(q.jobs, job)
	q.cond.Signal()
	q.mu.Unlock()

	select {
	case res := <-job.result:
		return res, nil
	case <-ctx.Done():
		q.remove(job)
		return dryJobResult{}, ctx.Err()
	}
}

func (q *dryJobQueue) remove(target *dryJob) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, job := range q.jobs {
		if job == target {
			q.jobs = append(q.jobs[:i], q.jobs[i+1:]...)
			return
		}
	}
}

func (q *dryJobQueue) Close() {
	q.mu.Lock()
	q.closed = true
	q.cond.Broadcast()
	q.mu.Unlock()
}
