package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tptconduit/engine"
)

func newTestEngine(t *testing.T) *engine.Engine {
	t.Helper()
	e := engine.NewEngine(engine.NewInMemoryEventLog(), engine.NewInMemoryStore(), 0)
	wf := engine.WorkflowDef{
		Name:    "signoff",
		Version: "v1",
		Initial: "approve",
		Steps: []engine.StepDef{{
			Name: "approve",
			Kind: engine.KindApproval,
			Approval: &engine.ApprovalDef{
				Chain: []engine.Approver{{Role: "manager"}, {Role: "director"}},
			},
		}},
	}
	if err := e.RegisterWorkflow(wf); err != nil {
		t.Fatalf("register workflow: %v", err)
	}
	return e
}

func gqlPost(t *testing.T, srv *Server, auth string, query string, vars map[string]any) (int, map[string]any) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"query": query, "variables": vars})
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	if auth != "" {
		req.Header.Set("X-API-Key", auth)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	var out map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
	}
	return rec.Code, out
}

func TestAuthEnforcement(t *testing.T) {
	e := newTestEngine(t)
	srv, err := NewServer(e, AuthConfig{Username: "admin", Password: "secret", APIKeys: []string{"key-123"}})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	q := `{ __typename }`
	// No credentials -> 401.
	code, _ := gqlPost(t, srv, "", q, nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", code)
	}
	// Valid API key -> 200.
	code, _ = gqlPost(t, srv, "key-123", q, nil)
	if code != http.StatusOK {
		t.Fatalf("expected 200 with api key, got %d", code)
	}
	// Valid basic auth -> 200.
	body, _ := json.Marshal(map[string]any{"query": q})
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.SetBasicAuth("admin", "secret")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with basic auth, got %d", rec.Code)
	}
}

func TestCreateTicketQueryAndApprove(t *testing.T) {
	e := newTestEngine(t)
	srv, err := NewServer(e, AuthConfig{APIKeys: []string{"k"}})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	// Create a ticket.
	create := `mutation($in: CreateTicketInput!) { createTicket(input: $in) { id workflow workflowVersion title } }`
	_, out := gqlPost(t, srv, "k", create, map[string]any{
		"in": map[string]any{"workflow": "signoff", "version": "v1", "title": "Need signoff", "fields": map[string]any{"amount": 100}},
	})
	data, ok := out["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data in response: %v", out)
	}
	tk, ok := data["createTicket"].(map[string]any)
	if !ok {
		t.Fatalf("no createTicket in response: %v", out)
	}
	if tk["title"] != "Need signoff" {
		t.Fatalf("unexpected title: %v", tk)
	}

	// List tickets.
	_, out = gqlPost(t, srv, "k", `{ tickets { id title fields } }`, nil)
	data = out["data"].(map[string]any)
	tickets := data["tickets"].([]any)
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
	fields := tickets[0].(map[string]any)["fields"].(map[string]any)
	if fields["amount"] != float64(100) {
		t.Fatalf("expected amount=100, got %v", fields["amount"])
	}

	// Find the run and approve the first link.
	_, out = gqlPost(t, srv, "k", `{ runs { id steps { name status approval { status index } } } }`, nil)
	data = out["data"].(map[string]any)
	runs := data["runs"].([]any)
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	run := runs[0].(map[string]any)
	runID := run["id"].(string)

	_, out = gqlPost(t, srv, "k",
		`mutation { approve(runId: "RUN", step: "approve", by: "manager") }`,
		nil)
	// runId is dynamic; substitute via variables instead.
	_, out = gqlPost(t, srv, "k",
		`mutation($runId: String!, $step: String!) { approve(runId: $runId, step: $step, by: "manager") }`,
		map[string]any{"runId": runID, "step": "approve"})
	if errs, ok := out["errors"].([]any); ok && len(errs) > 0 {
		t.Fatalf("approve first link failed: %v", errs)
	}

	// Second link.
	_, out = gqlPost(t, srv, "k",
		`mutation($runId: String!, $step: String!) { approve(runId: $runId, step: $step, by: "director") }`,
		map[string]any{"runId": runID, "step": "approve"})
	if errs, ok := out["errors"].([]any); ok && len(errs) > 0 {
		t.Fatalf("approve second link failed: %v", errs)
	}

	// Run should now be COMPLETED.
	_, out = gqlPost(t, srv, "k", `query($id: String!) { run(id: $id) { id status } }`, map[string]any{"id": runID})
	data = out["data"].(map[string]any)
	got := data["run"].(map[string]any)["status"]
	if got != "COMPLETED" {
		t.Fatalf("expected run COMPLETED, got %v", got)
	}
}

func TestWorkflowAndEventQueries(t *testing.T) {
	e := newTestEngine(t)
	srv, _ := NewServer(e, AuthConfig{APIKeys: []string{"k"}})
	_, out := gqlPost(t, srv, "k", `{ workflows { name version steps { name kind } } }`, nil)
	data := out["data"].(map[string]any)
	wfs := data["workflows"].([]any)
	if len(wfs) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(wfs))
	}
	// Create a ticket so a run exists, then read its timeline.
	_, out = gqlPost(t, srv, "k",
		`mutation($in: CreateTicketInput!) { createTicket(input: $in) { id } }`,
		map[string]any{"in": map[string]any{"workflow": "signoff", "version": "v1", "title": "x"}})
	runID := out["data"].(map[string]any)["createTicket"].(map[string]any)["id"].(string)
	// id is the ticket id, not run id; resolve run id.
	_, out = gqlPost(t, srv, "k", `{ runs { id } }`, nil)
	runID = out["data"].(map[string]any)["runs"].([]any)[0].(map[string]any)["id"].(string)
	_, out = gqlPost(t, srv, "k", `query($id: String!) { events(runId: $id) { seq type } }`, map[string]any{"id": runID})
	data = out["data"].(map[string]any)
	events := data["events"].([]any)
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}
}
