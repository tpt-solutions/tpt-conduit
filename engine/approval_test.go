package engine_test

import (
	"context"
	"testing"
	"time"

	"tptconduit/engine"
	"tptconduit/engine/examples"
)

func TestApprovalChain(t *testing.T) {
	log := engine.NewInMemoryEventLog()
	store := engine.NewInMemoryStore()
	eng := engine.NewEngine(log, store, 1)
	defer eng.Close()
	_ = eng.RegisterWorkflow(examples.ApprovalChain())
	eng.RegisterTask("open_request", okTask{}.Handler())
	eng.RegisterTask("provision_resource", okTask{}.Handler())
	eng.RegisterTask("send_notification", okTask{}.Handler())
	eng.RegisterTask("record_rejection", okTask{}.Handler())

	_, run, err := eng.CreateTicket(context.Background(), "generic-approval-chain", "1.0.0", "Buy laptop", nil)
	if err != nil {
		t.Fatal(err)
	}
	run = waitFor(t, eng, run.ID, func(r *engine.WorkflowRun) bool {
		s := r.Steps["review"]
		return s != nil && s.Status == engine.StepWaiting
	}, 2*time.Second)
	if run.Steps["review"].Status != engine.StepWaiting {
		t.Fatalf("expected review waiting, got %s", run.Steps["review"].Status)
	}
	// Approval state should be at link 0.
	if run.Steps["review"].Approval.Index != 0 {
		t.Fatalf("expected approval index 0, got %d", run.Steps["review"].Approval.Index)
	}
	if len(run.Steps["review"].Approval.Chain) != 3 {
		t.Fatalf("expected chain length 3, got %d", len(run.Steps["review"].Approval.Chain))
	}

	// Approve first two links; run stays waiting until the last.
	_ = eng.Approve(context.Background(), run.ID, "review", "manager", "")
	run, _ = eng.GetRun(context.Background(), run.ID)
	if run.Steps["review"].Approval.Index != 1 {
		t.Fatalf("expected index advanced to 1, got %d", run.Steps["review"].Approval.Index)
	}
	_ = eng.Approve(context.Background(), run.ID, "review", "director", "")
	run, _ = eng.GetRun(context.Background(), run.ID)
	if run.Steps["review"].Approval.Index != 2 {
		t.Fatalf("expected index advanced to 2, got %d", run.Steps["review"].Approval.Index)
	}
	// Approve the final link -> step completes and provision runs.
	_ = eng.Approve(context.Background(), run.ID, "review", "finance", "")
	run = waitFor(t, eng, run.ID, func(r *engine.WorkflowRun) bool {
		return r.Status == engine.RunStatusCompleted
	}, 2*time.Second)
	if run.Status != engine.RunStatusCompleted {
		t.Fatalf("expected completed after full approval, got %s (steps=%v)", run.Status, statuses(run))
	}
	if run.Steps["review"].Status != engine.StepCompleted {
		t.Fatalf("expected review completed, got %s", run.Steps["review"].Status)
	}
}

func TestApprovalRejection(t *testing.T) {
	log := engine.NewInMemoryEventLog()
	store := engine.NewInMemoryStore()
	eng := engine.NewEngine(log, store, 1)
	defer eng.Close()
	_ = eng.RegisterWorkflow(examples.ApprovalChain())
	eng.RegisterTask("open_request", okTask{}.Handler())
	eng.RegisterTask("provision_resource", okTask{}.Handler())
	eng.RegisterTask("send_notification", okTask{}.Handler())
	eng.RegisterTask("record_rejection", okTask{}.Handler())

	_, run, err := eng.CreateTicket(context.Background(), "generic-approval-chain", "1.0.0", "Buy laptop", nil)
	if err != nil {
		t.Fatal(err)
	}
	run = waitFor(t, eng, run.ID, func(r *engine.WorkflowRun) bool {
		return r.Steps["review"] != nil && r.Steps["review"].Status == engine.StepWaiting
	}, 2*time.Second)
	_ = eng.Reject(context.Background(), run.ID, "review", "manager", "too expensive")
	run = waitFor(t, eng, run.ID, func(r *engine.WorkflowRun) bool {
		return r.Status == engine.RunStatusFailed
	}, 2*time.Second)
	if run.Status != engine.RunStatusFailed {
		t.Fatalf("expected failed run after rejection, got %s", run.Status)
	}
}

func TestRoutingRules(t *testing.T) {
	r := engine.NewRouter()
	ticket := &engine.Ticket{Fields: map[string]any{"category": "hardware", "impact": "high"}}

	rules := []engine.RoutingRule{
		{If: map[string]any{"category": "hardware"}, Queue: "it-hardware", Priority: "high"},
		{If: map[string]any{"category": "software"}, Queue: "it-desk"},
	}
	r.Apply(ticket, rules)
	if ticket.Queue != "it-hardware" {
		t.Fatalf("expected it-hardware, got %q", ticket.Queue)
	}
	if ticket.Priority != "high" {
		t.Fatalf("expected high, got %q", ticket.Priority)
	}

	// No match keeps defaults.
	t2 := &engine.Ticket{Fields: map[string]any{"category": "unknown"}}
	r.Apply(t2, rules)
	if t2.Queue != "" {
		t.Fatalf("expected empty queue, got %q", t2.Queue)
	}
}

func TestDSLParse(t *testing.T) {
	def := examples.Helpdesk()
	if def.Name != "it-helpdesk" {
		t.Fatalf("unexpected name %q", def.Name)
	}
	if def.Initial != "triage" {
		t.Fatalf("unexpected initial %q", def.Initial)
	}
	// Verify a delay step parses its duration in the asset template.
	asset := examples.AssetTracking()
	var wait engine.StepDef
	found := false
	for _, s := range asset.Steps {
		if s.Name == "return_wait" {
			wait = s
			found = true
		}
	}
	if !found {
		t.Fatal("return_wait step not found")
	}
	if wait.Kind != engine.KindDelay || wait.Delay == nil {
		t.Fatalf("return_wait should be a delay step")
	}
	if wait.Delay.Duration != 48*time.Hour {
		t.Fatalf("expected 48h delay, got %v", wait.Delay.Duration)
	}
}
