package engine_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tptconduit/engine"
	"tptconduit/engine/examples"
)

// TestPostgresBackend exercises the Postgres event log + store. It is skipped
// unless PG_TEST=1 and DATABASE_URL point at a reachable Postgres instance,
// because CI may not have one. Run locally with:
//
//	PG_TEST=1 DATABASE_URL='postgres://user:pass@localhost:5432/conduit?sslmode=disable' go test ./engine -run Postgres
func TestPostgresBackend(t *testing.T) {
	if os.Getenv("PG_TEST") != "1" {
		t.Skip("set PG_TEST=1 and DATABASE_URL to run Postgres integration tests")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	log := engine.NewPostgresEventLog(db)
	store := engine.NewPostgresStore(db)
	eng := engine.NewEngine(log, store, 2)
	defer eng.Close()
	if err := eng.RegisterWorkflow(examples.Helpdesk()); err != nil {
		t.Fatal(err)
	}
	eng.RegisterTask("triage_ticket", okTask{}.Handler())
	eng.RegisterTask("assign_owner", okTask{}.Handler())
	eng.RegisterTask("resolve_ticket", okTask{}.Handler())
	eng.RegisterTask("close_ticket", okTask{}.Handler())
	eng.RegisterTask("escalate_ticket", okTask{}.Handler())
	eng.RegisterTask("mark_failed", okTask{}.Handler())

	_, run, err := eng.CreateTicket(context.Background(), "it-helpdesk", "1.0.0", "PG ticket", map[string]any{"category": "email"})
	if err != nil {
		t.Fatal(err)
	}
	run = waitFor(t, eng, run.ID, func(r *engine.WorkflowRun) bool {
		return r.Status == engine.RunStatusCompleted
	}, 5*time.Second)
	if run.Status != engine.RunStatusCompleted {
		t.Fatalf("pg: expected completed, got %s (steps=%v)", run.Status, statuses(run))
	}

	// Confirm durability: a fresh engine over the same database replays exactly.
	eng2 := engine.NewEngine(log, store, 2)
	defer eng2.Close()
	replayed, err := eng2.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Status != engine.RunStatusCompleted {
		t.Fatalf("pg replay: expected completed, got %s", replayed.Status)
	}
}
