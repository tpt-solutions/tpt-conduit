package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresEventLog is a durable EventLog backed by PostgreSQL. It relies on the
// schema in schema.sql: a single append-only `events` table with a per-run
// sequence enforced by a unique (run_id, seq) constraint.
type PostgresEventLog struct {
	db *sql.DB
}

// NewPostgresEventLog opens a PostgreSQL-backed event log.
func NewPostgresEventLog(db *sql.DB) *PostgresEventLog {
	return &PostgresEventLog{db: db}
}

func (l *PostgresEventLog) Append(ctx context.Context, e Event) (Event, error) {
	var sched sql.NullTime
	if e.ScheduleAt != nil {
		sched = sql.NullTime{Time: *e.ScheduleAt, Valid: true}
	}
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return Event{}, fmt.Errorf("append event: begin tx: %w", err)
	}
	defer tx.Rollback()

	// Atomically reserve the next seq for this run. The INSERT ... ON CONFLICT
	// DO UPDATE takes a row lock on the run's counter row, so concurrent
	// appends to the same run_id are serialized and never collide.
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO event_seq (run_id, next_seq) VALUES ($1, 2)
		 ON CONFLICT (run_id) DO UPDATE SET next_seq = event_seq.next_seq + 1
		 RETURNING next_seq - 1`,
		e.RunID,
	).Scan(&seq); err != nil {
		return Event{}, fmt.Errorf("append event: reserve seq: %w", err)
	}

	var at time.Time
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO events (run_id, seq, ticket_id, type, at, payload, schedule_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING at`,
		e.RunID, seq, e.TicketID, string(e.Type), e.At, e.Payload, sched,
	).Scan(&at); err != nil {
		return Event{}, fmt.Errorf("append event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Event{}, fmt.Errorf("append event: commit: %w", err)
	}

	e.Seq = seq
	if !e.At.IsZero() {
		e.At = at
	}
	if e.ID == 0 {
		e.ID = seq
	}
	return e, nil
}

func (l *PostgresEventLog) History(ctx context.Context, runID string) ([]Event, error) {
	rows, err := l.db.QueryContext(ctx,
		`SELECT seq, ticket_id, type, at, payload, schedule_at
		 FROM events WHERE run_id=$1 ORDER BY seq ASC`, runID)
	if err != nil {
		return nil, fmt.Errorf("history: %w", err)
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var typ string
		var payload []byte
		var sched sql.NullTime
		if err := rows.Scan(&e.Seq, &e.TicketID, &typ, &e.At, &payload, &sched); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.RunID = runID
		e.Type = EventType(typ)
		e.Payload = payload
		if sched.Valid {
			t := sched.Time
			e.ScheduleAt = &t
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (l *PostgresEventLog) Runs(ctx context.Context) ([]string, error) {
	rows, err := l.db.QueryContext(ctx, `SELECT DISTINCT run_id FROM events ORDER BY run_id`)
	if err != nil {
		return nil, fmt.Errorf("runs: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// Store persists the non-event-sourced projection data: tickets and the
// currently deployed workflow definitions.
type Store interface {
	SaveTicket(ctx context.Context, t Ticket) error
	GetTicket(ctx context.Context, id string) (Ticket, error)
	ListTickets(ctx context.Context) ([]Ticket, error)
	RegisterWorkflow(ctx context.Context, w WorkflowDef) error
	GetWorkflow(ctx context.Context, name, version string) (WorkflowDef, error)
	ListWorkflows(ctx context.Context) ([]WorkflowDef, error)
	WorkflowVersions(ctx context.Context, name string) ([]string, error)
}

// PostgresStore is the PostgreSQL-backed Store.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore opens a PostgreSQL-backed store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) SaveTicket(ctx context.Context, t Ticket) error {
	fields, err := json.Marshal(t.Fields)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO tickets (id, workflow, workflow_version, title, fields, assignee, queue, priority, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 ON CONFLICT (id) DO UPDATE SET
		   workflow=EXCLUDED.workflow, workflow_version=EXCLUDED.workflow_version,
		   title=EXCLUDED.title, fields=EXCLUDED.fields, assignee=EXCLUDED.assignee,
		   queue=EXCLUDED.queue, priority=EXCLUDED.priority, updated_at=EXCLUDED.updated_at`,
		t.ID, t.Workflow, t.WorkflowVer, t.Title, fields, t.Assignee, t.Queue, t.Priority, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetTicket(ctx context.Context, id string) (Ticket, error) {
	var t Ticket
	var fields []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT id, workflow, workflow_version, title, fields, assignee, queue, priority, created_at, updated_at
		 FROM tickets WHERE id=$1`, id).
		Scan(&t.ID, &t.Workflow, &t.WorkflowVer, &t.Title, &fields, &t.Assignee, &t.Queue, &t.Priority, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Ticket{}, ErrNotFound
	}
	if err != nil {
		return Ticket{}, fmt.Errorf("get ticket: %w", err)
	}
	if len(fields) > 0 {
		_ = json.Unmarshal(fields, &t.Fields)
	}
	return t, nil
}

func (s *PostgresStore) RegisterWorkflow(ctx context.Context, w WorkflowDef) error {
	body, err := json.Marshal(w)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO workflows (name, version, def) VALUES ($1,$2,$3)
		 ON CONFLICT (name, version) DO UPDATE SET def=EXCLUDED.def`,
		w.Name, w.Version, body)
	if err != nil {
		return fmt.Errorf("register workflow: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetWorkflow(ctx context.Context, name, version string) (WorkflowDef, error) {
	var body []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT def FROM workflows WHERE name=$1 AND version=$2`, name, version).
		Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkflowDef{}, ErrNotFound
	}
	if err != nil {
		return WorkflowDef{}, fmt.Errorf("get workflow: %w", err)
	}
	var w WorkflowDef
	if err := json.Unmarshal(body, &w); err != nil {
		return WorkflowDef{}, err
	}
	return w, nil
}

func (s *PostgresStore) WorkflowVersions(ctx context.Context, name string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT version FROM workflows WHERE name=$1 ORDER BY version`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *PostgresStore) ListTickets(ctx context.Context) ([]Ticket, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, workflow, workflow_version, title, fields, assignee, queue, priority, created_at, updated_at
		 FROM tickets ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	defer rows.Close()
	var out []Ticket
	for rows.Next() {
		var t Ticket
		var fields []byte
		if err := rows.Scan(&t.ID, &t.Workflow, &t.WorkflowVer, &t.Title, &fields, &t.Assignee, &t.Queue, &t.Priority, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan ticket: %w", err)
		}
		if len(fields) > 0 {
			_ = json.Unmarshal(fields, &t.Fields)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *PostgresStore) ListWorkflows(ctx context.Context) ([]WorkflowDef, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT def FROM workflows ORDER BY name, version`)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	defer rows.Close()
	var out []WorkflowDef
	for rows.Next() {
		var body []byte
		if err := rows.Scan(&body); err != nil {
			return nil, fmt.Errorf("scan workflow: %w", err)
		}
		var w WorkflowDef
		if err := json.Unmarshal(body, &w); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
