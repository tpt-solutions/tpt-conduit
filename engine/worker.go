package engine

import (
	"context"
	"sync"
)

// taskItem is a unit of automated work pulled from the queue by a worker.
type taskItem struct {
	RunID    string
	TicketID string
	Step     string
}

// TaskExecutor runs a single task item. It is supplied by the Engine.
type TaskExecutor func(ctx context.Context, t taskItem) error

// WorkerPool is an in-process, durable-by-replay task queue. Tasks are pulled
// from an in-memory channel; if the process crashes mid-execution, the task is
// simply re-derived and re-enqueued during Engine.Recover, because the engine
// only marks a step complete via an append-only event.
type WorkerPool struct {
	tasks   chan taskItem
	exec    TaskExecutor
	wg      sync.WaitGroup
	closeCh chan struct{}
	once    sync.Once
}

// NewWorkerPool creates a pool with the given worker count and executor. A
// count of 0 yields a pool that accepts submissions but never executes them
// (useful for tests that drive steps deterministically).
func NewWorkerPool(workers int, exec TaskExecutor) *WorkerPool {
	p := &WorkerPool{
		tasks:   make(chan taskItem, 1024),
		exec:    exec,
		closeCh: make(chan struct{}),
	}
	if workers < 0 {
		workers = 0
	}
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.loop()
	}
	return p
}

func (p *WorkerPool) loop() {
	defer p.wg.Done()
	for {
		select {
		case <-p.closeCh:
			return
		case t, ok := <-p.tasks:
			if !ok {
				return
			}
			if err := p.exec(context.Background(), t); err != nil {
				// Errors are recorded as step failures inside the executor;
				// nothing else to do here. The event log is the source of truth.
				_ = err
			}
		}
	}
}

// Submit enqueues a task for execution.
func (p *WorkerPool) Submit(t taskItem) {
	select {
	case p.tasks <- t:
	case <-p.closeCh:
	}
}

// Stop gracefully drains and stops all workers.
func (p *WorkerPool) Stop() error {
	p.once.Do(func() {
		close(p.closeCh)
	})
	p.wg.Wait()
	return nil
}
