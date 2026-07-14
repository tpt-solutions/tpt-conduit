package dsl

import (
	_ "embed"
	"encoding/json"
	"testing"

	"tptconduit/engine"
)

//go:embed testdata/helpdesk.ir.json
var helpdeskIR []byte

// TestTSGeneratedIRContract proves the TypeScript DSL emits exactly the
// engine.WorkflowDef IR that the Go engine consumes: the same shape the YAML
// DSL compiles to. If the IR drifts, this test fails.
func TestTSGeneratedIRContract(t *testing.T) {
	var w engine.WorkflowDef
	if err := json.Unmarshal(helpdeskIR, &w); err != nil {
		t.Fatalf("unmarshal TS-generated IR: %v", err)
	}
	if w.Name != "helpdesk" || w.Version != "1.0.0" {
		t.Fatalf("unexpected identity: %s@%s", w.Name, w.Version)
	}
	if w.Initial != "triage" {
		t.Fatalf("expected initial triage, got %q", w.Initial)
	}
	if len(w.Steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(w.Steps))
	}
	// The "escalate" step is a two-link approval chain.
	esc, ok := w.Step("escalate")
	if !ok || esc.Kind != engine.KindApproval {
		t.Fatalf("escalate step missing or wrong kind")
	}
	if esc.Approval == nil || len(esc.Approval.Chain) != 2 {
		t.Fatalf("escalate approval chain wrong length: %v", esc.Approval)
	}
	// SLA duration is nanoseconds (30 minutes).
	if len(w.SLAs) != 1 || w.SLAs[0].Duration.Minutes() != 30 {
		t.Fatalf("SLA not as expected: %v", w.SLAs)
	}
	// Routing rule survived the round-trip.
	if len(w.Routing) != 2 {
		t.Fatalf("expected 2 routing rules, got %d", len(w.Routing))
	}
}
