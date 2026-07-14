package engine

import (
	"context"
	"sync"
	"time"
)

// EventLog is the append-only, durable history of every workflow run. State is
// never mutated in place; it is always derived by replaying these events.
type EventLog interface {
	// Append records a new event. The implementation is responsible for
	// assigning a monotonic sequence number per run and persisting atomically.
	Append(ctx context.Context, e Event) (Event, error)
	// History returns all events for a run in sequence order.
	History(ctx context.Context, runID string) ([]Event, error)
	// Runs returns the IDs of all known runs (used for crash recovery).
	Runs(ctx context.Context) ([]string, error)
}

// InMemoryEventLog is a thread-safe, non-durable EventLog used for tests and
// single-process development. It is the reference implementation that the
// Postgres-backed log mirrors exactly.
type InMemoryEventLog struct {
	mu     sync.Mutex
	seq    map[string]int64
	events map[string][]Event
	runs   map[string]struct{}
}

// NewInMemoryEventLog creates an empty in-memory event log.
func NewInMemoryEventLog() *InMemoryEventLog {
	return &InMemoryEventLog{
		seq:    map[string]int64{},
		events: map[string][]Event{},
		runs:   map[string]struct{}{},
	}
}

func (l *InMemoryEventLog) Append(ctx context.Context, e Event) (Event, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e.Seq = l.seq[e.RunID] + 1
	l.seq[e.RunID]++
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	if e.ID == 0 {
		e.ID = e.Seq
	}
	cp := e
	l.events[e.RunID] = append(l.events[e.RunID], cp)
	l.runs[e.RunID] = struct{}{}
	return cp, nil
}

func (l *InMemoryEventLog) History(ctx context.Context, runID string) ([]Event, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	src := l.events[runID]
	out := make([]Event, len(src))
	copy(out, src)
	return out, nil
}

func (l *InMemoryEventLog) Runs(ctx context.Context) ([]string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, 0, len(l.runs))
	for id := range l.runs {
		out = append(out, id)
	}
	return out, nil
}
