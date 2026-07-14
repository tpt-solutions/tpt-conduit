package engine_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"tptconduit/engine"
	"tptconduit/engine/examples"
)

// newTestEngine wires an in-memory log/store with a worker pool and registers
// the helpdesk workflow plus trivial task handlers that always succeed.
func newTestEngine(t *testing.T) (*engine.Engine, func(string) map[string]any) {
	t.Helper()
	log := engine.NewInMemoryEventLog()
	store := engine.NewInMemoryStore()
	eng := engine.NewEngine(log, store, 4)

	if err := eng.RegisterWorkflow(examples.Helpdesk()); err != nil {
		t.Fatalf("register helpdesk: %v", err)
	}

	var mu sync.Mutex
	outputs := map[string]map[string]any{}
	eng.RegisterTask("triage_ticket", func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return map[string]any{"triaged": true}, nil
	})
	eng.RegisterTask("assign_owner", func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return map[string]any{"owner": "alice"}, nil
	})
	eng.RegisterTask("resolve_ticket", func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		mu.Lock()
		outputs[step] = map[string]any{"resolved": true}
		mu.Unlock()
		return map[string]any{"resolved": true}, nil
	})
	eng.RegisterTask("close_ticket", func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return map[string]any{"closed": true}, nil
	})
	eng.RegisterTask("escalate_ticket", func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return map[string]any{"escalated": true}, nil
	})
	eng.RegisterTask("mark_failed", func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return nil, nil
	})
	return eng, func(step string) map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return outputs[step]
	}
}

func waitFor(t *testing.T, eng *engine.Engine, runID string, pred func(*engine.WorkflowRun) bool, timeout time.Duration) *engine.WorkflowRun {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		run, err := eng.GetRun(context.Background(), runID)
		if err == nil && pred(run) {
			return run
		}
		time.Sleep(5 * time.Millisecond)
	}
	run, _ := eng.GetRun(context.Background(), runID)
	return run
}

func TestCreateTicketStartsRun(t *testing.T) {
	eng, _ := newTestEngine(t)
	defer eng.Close()

	ticket, run, err := eng.CreateTicket(context.Background(), "it-helpdesk", "1.0.0", "Laptop broken", map[string]any{"category": "hardware"})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if ticket.ID == "" || run.ID == "" {
		t.Fatal("expected non-empty ids")
	}
	if run.Status != engine.RunStatusActive {
		t.Fatalf("expected active run, got %s", run.Status)
	}
	// Routing should have placed hardware tickets in the hardware queue.
	if ticket.Queue != "it-hardware" {
		t.Fatalf("expected it-hardware queue, got %q", ticket.Queue)
	}
	if ticket.Priority != "high" {
		t.Fatalf("expected high priority, got %q", ticket.Priority)
	}
}

func TestHappyPathCompletes(t *testing.T) {
	eng, _ := newTestEngine(t)
	defer eng.Close()

	_, run, err := eng.CreateTicket(context.Background(), "it-helpdesk", "1.0.0", "Need VPN", map[string]any{"category": "access"})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	run = waitFor(t, eng, run.ID, func(r *engine.WorkflowRun) bool {
		return r.Status == engine.RunStatusCompleted
	}, 2*time.Second)
	if run.Status != engine.RunStatusCompleted {
		t.Fatalf("expected completed, got %s (steps=%v)", run.Status, statuses(run))
	}
	if run.Steps["close"].Status != engine.StepCompleted {
		t.Fatalf("expected close step completed, got %s", run.Steps["close"].Status)
	}
}

func TestTaskFailureTriggersOnError(t *testing.T) {
	log := engine.NewInMemoryEventLog()
	store := engine.NewInMemoryStore()
	eng := engine.NewEngine(log, store, 2)
	defer eng.Close()
	if err := eng.RegisterWorkflow(examples.Helpdesk()); err != nil {
		t.Fatal(err)
	}
	eng.RegisterTask("triage_ticket", func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return nil, nil
	})
	eng.RegisterTask("assign_owner", func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return nil, nil
	})
	// resolve always fails -> jumps to `failed` step.
	eng.RegisterTask("resolve_ticket", func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return nil, context.DeadlineExceeded
	})
	eng.RegisterTask("escalate_ticket", func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return nil, nil
	})
	eng.RegisterTask("close_ticket", func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return nil, nil
	})
	eng.RegisterTask("mark_failed", func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return nil, nil
	})

	_, run, err := eng.CreateTicket(context.Background(), "it-helpdesk", "1.0.0", "Broken", map[string]any{"category": "email"})
	if err != nil {
		t.Fatal(err)
	}
	run = waitFor(t, eng, run.ID, func(r *engine.WorkflowRun) bool {
		return r.Status == engine.RunStatusFailed
	}, 2*time.Second)
	if run.Status != engine.RunStatusFailed {
		t.Fatalf("expected failed run, got %s steps=%v", run.Status, statuses(run))
	}
	if run.Steps["failed"].Status != engine.StepCompleted {
		t.Fatalf("expected failed step completed, got %s", run.Steps["failed"].Status)
	}
}

func statuses(r *engine.WorkflowRun) map[string]engine.StepStatus {
	out := map[string]engine.StepStatus{}
	for k, v := range r.Steps {
		out[k] = v.Status
	}
	return out
}
