package engine_test

import (
	"context"
	"testing"
	"time"

	"tptconduit/engine"
	"tptconduit/engine/examples"
)

// TestCrashRecovery simulates a process crash by throwing away the in-memory
// engine (and its worker queue) and rebuilding from the same event log. The
// recovered run must resume exactly where it left off and finish identically.
func TestCrashRecovery(t *testing.T) {
	log := engine.NewInMemoryEventLog()
	store := engine.NewInMemoryStore()

	// First "process": create a ticket and let it partially execute.
	eng1 := engine.NewEngine(log, store, 2)
	_ = eng1.RegisterWorkflow(examples.Helpdesk())
	eng1.RegisterTask("triage_ticket", okTask{"triaged": true}.Handler())
	eng1.RegisterTask("assign_owner", okTask{"owner": "bob"}.Handler())
	eng1.RegisterTask("resolve_ticket", okTask{"resolved": true}.Handler())
	eng1.RegisterTask("close_ticket", okTask{"closed": true}.Handler())
	eng1.RegisterTask("escalate_ticket", okTask{}.Handler())
	eng1.RegisterTask("mark_failed", okTask{}.Handler())

	_, run, err := eng1.CreateTicket(context.Background(), "it-helpdesk", "1.0.0", "Crash me", map[string]any{"category": "email"})
	if err != nil {
		t.Fatal(err)
	}
	// Wait until triage + assign have run, then "crash" before close.
	run = waitFor(t, eng1, run.ID, func(r *engine.WorkflowRun) bool {
		if r == nil {
			return false
		}
		s := r.Steps["assign"]
		return s != nil && s.Status == engine.StepCompleted
	}, 2*time.Second)
	if run == nil || run.Steps["assign"] == nil || run.Steps["assign"].Status != engine.StepCompleted {
		t.Fatalf("precondition failed: assign not done (run=%v)", run)
	}
	_ = eng1.Close() // crash: worker queue discarded

	// Second "process": brand new engine, same durable log + store.
	eng2 := engine.NewEngine(log, store, 2)
	_ = eng2.RegisterWorkflow(examples.Helpdesk())
	eng2.RegisterTask("triage_ticket", okTask{"triaged": true}.Handler())
	eng2.RegisterTask("assign_owner", okTask{"owner": "bob"}.Handler())
	eng2.RegisterTask("resolve_ticket", okTask{"resolved": true}.Handler())
	eng2.RegisterTask("close_ticket", okTask{"closed": true}.Handler())
	eng2.RegisterTask("escalate_ticket", okTask{}.Handler())
	eng2.RegisterTask("mark_failed", okTask{}.Handler())

	if err := eng2.Recover(context.Background()); err != nil {
		t.Fatalf("recover: %v", err)
	}
	run = waitFor(t, eng2, run.ID, func(r *engine.WorkflowRun) bool {
		return r.Status == engine.RunStatusCompleted
	}, 2*time.Second)
	if run.Status != engine.RunStatusCompleted {
		t.Fatalf("after recovery expected completed, got %s (steps=%v)", run.Status, statuses(run))
	}
	// Replaying the same log must be deterministic: resolved flag present.
	if run.Steps["resolve"].Status != engine.StepCompleted {
		t.Fatalf("resolve not completed after recovery")
	}
}

// TestReplayDeterminism verifies the core invariant: the same event stream
// always yields identical state, regardless of the engine instance.
func TestReplayDeterminism(t *testing.T) {
	log := engine.NewInMemoryEventLog()
	store := engine.NewInMemoryStore()

	// Manually record a run's events via one engine, then replay with another.
	eng1 := engine.NewEngine(log, store, 0)
	_ = eng1.RegisterWorkflow(examples.Helpdesk())

	_, run, _ := eng1.CreateTicket(context.Background(), "it-helpdesk", "1.0.0", "Det", map[string]any{"category": "software"})

	eng2 := engine.NewEngine(log, store, 0)
	_ = eng2.RegisterWorkflow(examples.Helpdesk())
	run2, err := eng2.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Steps) != len(run2.Steps) {
		t.Fatalf("step count mismatch: %d vs %d", len(run.Steps), len(run2.Steps))
	}
	for k, v := range run.Steps {
		v2, ok := run2.Steps[k]
		if !ok {
			t.Fatalf("step %q missing on replay", k)
		}
		if v.Status != v2.Status {
			t.Fatalf("step %q status drift: %s vs %s", k, v.Status, v2.Status)
		}
	}
}

// okTask is a trivial always-succeeding handler keyed by name.
type okTask map[string]any

func (o okTask) run(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
	return map[string]any(o), nil
}

// adapt okTask to the TaskHandler signature used by RegisterTask.
func (o okTask) Handler() engine.TaskHandler {
	return func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return o.run(ctx, run, step, fields)
	}
}

func TestDelayStepFires(t *testing.T) {
	log := engine.NewInMemoryEventLog()
	store := engine.NewInMemoryStore()
	eng := engine.NewEngine(log, store, 1)
	defer eng.Close()
	_ = eng.RegisterWorkflow(examples.HROnboarding())
	eng.RegisterTask("create_employee_record", okTask{}.Handler())
	eng.RegisterTask("provision_accounts", okTask{}.Handler())
	eng.RegisterTask("ship_equipment", okTask{}.Handler())
	eng.RegisterTask("send_welcome", okTask{}.Handler())
	eng.RegisterTask("escalate_onboarding", okTask{}.Handler())
	eng.RegisterTask("mark_failed", okTask{}.Handler())

	_, run, err := eng.CreateTicket(context.Background(), "hr-onboarding", "1.0.0", "Hire Eve", nil)
	if err != nil {
		t.Fatal(err)
	}
	// After create + approval (no approval steps pending since none requested?...)
	// HR onboarding's second step is an approval; auto-approve it.
	run = waitFor(t, eng, run.ID, func(r *engine.WorkflowRun) bool {
		return r.Steps["manager_approval"] != nil && r.Steps["manager_approval"].Status == engine.StepWaiting
	}, 2*time.Second)
	if run.Steps["manager_approval"].Status != engine.StepWaiting {
		t.Fatalf("expected manager_approval waiting, got %s", run.Steps["manager_approval"].Status)
	}
	_ = eng.Approve(context.Background(), run.ID, "manager_approval", "mgr", "")
	_ = eng.Approve(context.Background(), run.ID, "manager_approval", "hrbp", "")
	// Now provision_accounts runs, then equipment_wait (delay 24h) then ship.
	run = waitFor(t, eng, run.ID, func(r *engine.WorkflowRun) bool {
		return r.Steps["equipment_wait"] != nil && r.Steps["equipment_wait"].Status == engine.StepWaiting
	}, 2*time.Second)
	if run.Steps["equipment_wait"].Status != engine.StepWaiting {
		t.Fatalf("expected equipment_wait waiting (delay), got %s", run.Steps["equipment_wait"].Status)
	}
}

func TestSLABreach(t *testing.T) {
	// Use a controlled clock so the SLA deadline is in the past immediately.
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	log := engine.NewInMemoryEventLog()
	store := engine.NewInMemoryStore()
	eng := engine.NewEngine(log, store, 0)
	eng.SetClock(func() time.Time { return now })
	_ = eng.RegisterWorkflow(examples.Helpdesk())

	_, run, err := eng.CreateTicket(context.Background(), "it-helpdesk", "1.0.0", "Slow", map[string]any{"category": "email"})
	if err != nil {
		t.Fatal(err)
	}
	// Advance clock beyond the 1h response SLA and re-check.
	eng.SetClock(func() time.Time { return now.Add(2 * time.Hour) })
	eng.CheckSLA(context.Background(), run, examples.Helpdesk())

	events, _ := log.History(context.Background(), run.ID)
	found := false
	for _, e := range events {
		if e.Type == engine.EventSLABreached {
			found = true
		}
	}
	if !found {
		t.Fatal("expected SLA breached event")
	}
}
