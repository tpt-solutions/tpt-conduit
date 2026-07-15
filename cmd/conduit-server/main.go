// Command conduit-server runs the TPT Conduit API: it wires the durable
// workflow engine behind the authenticated GraphQL endpoint, seeds the example
// workflow definitions, and listens for HTTP requests.
//
// Usage:
//
//	CONDUIT_ADDR=:8080 CONDUIT_USERNAME=admin CONDUIT_PASSWORD=secret \
//	  go run ./cmd/conduit-server
//
// With DATABASE_URL set it uses Postgres (applying engine/schema.sql); otherwise
// it runs fully in-memory (suitable for local dev and the e2e test suite).
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tptconduit/api"
	"tptconduit/engine"
	"tptconduit/engine/dsl"
)

func main() {
	addr := getenv("CONDUIT_ADDR", ":8080")
	username := getenv("CONDUIT_USERNAME", "admin")
	password := getenv("CONDUIT_PASSWORD", "secret")
	apiKeys := splitComma(getenv("CONDUIT_API_KEYS", ""))
	workflowsDir := getenv("CONDUIT_WORKFLOWS_DIR", "engine/examples")

	var e *engine.Engine
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			log.Fatalf("open postgres: %v", err)
		}
		defer db.Close()
		if err := applySchema(db); err != nil {
			log.Fatalf("apply schema: %v", err)
		}
		e = engine.NewEngine(engine.NewPostgresEventLog(db), engine.NewPostgresStore(db), 4)
	} else {
		e = engine.NewEngine(engine.NewInMemoryEventLog(), engine.NewInMemoryStore(), 4)
	}

	registry, err := dsl.LoadDir(workflowsDir)
	if err != nil {
		log.Fatalf("load workflows from %s: %v", workflowsDir, err)
	}
	if err := registry.RegisterAll(e); err != nil {
		log.Fatalf("register workflows: %v", err)
	}
	registerExampleTasks(e)
	log.Printf("loaded %d workflow definition(s) from %s", registry.Count(), workflowsDir)

	srv, err := api.NewServer(e, api.AuthConfig{Username: username, Password: password, APIKeys: apiKeys})
	if err != nil {
		log.Fatalf("build server: %v", err)
	}
	log.Printf("listening on %s", addr)
	log.Fatal(srv.Serve(addr))
}

func applySchema(db *sql.DB) error {
	schema, err := os.ReadFile("engine/schema.sql")
	if err != nil {
		return fmt.Errorf("read schema.sql: %w", err)
	}
	if _, err := db.ExecContext(context.Background(), string(schema)); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	return nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// registerExampleTasks binds no-op success handlers for the tasks used by the
// bundled example workflows (engine/examples). Without these the engine has no
// executor for a task step and the step fails, so the example flows can't
// progress to their approval steps. A real deployment overrides these with
// domain handlers; these exist so the demo + e2e suite run end-to-end.
func registerExampleTasks(e *engine.Engine) {
	tasks := []string{
		// generic-approval-chain
		"open_request", "provision_resource", "send_notification", "record_rejection",
		// asset_tracking
		"register_asset", "dispatch_asset", "remind_return", "mark_returned", "mark_failed",
		// helpdesk
		"triage_ticket", "assign_owner", "resolve_ticket", "escalate_ticket", "close_ticket",
		// hr_onboarding
		"create_employee_record", "provision_accounts", "ship_equipment",
		"escalate_onboarding", "send_welcome",
	}
	noop := func(ctx context.Context, run *engine.WorkflowRun, step string, fields map[string]any) (map[string]any, error) {
		return map[string]any{"step": step, "ok": true}, nil
	}
	for _, key := range tasks {
		e.RegisterTask(key, noop)
	}
}
