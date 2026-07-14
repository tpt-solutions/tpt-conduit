-- TPT Conduit — PostgreSQL schema (single-node, Phase 1).
-- The event log is the source of truth for workflow state; the tickets and
-- workflows tables are projections/indexes for fast lookup.

CREATE TABLE IF NOT EXISTS tickets (
    id               TEXT PRIMARY KEY,
    workflow         TEXT NOT NULL,
    workflow_version TEXT NOT NULL,
    title            TEXT NOT NULL,
    fields           JSONB NOT NULL DEFAULT '{}'::jsonb,
    assignee         TEXT NOT NULL DEFAULT '',
    queue            TEXT NOT NULL DEFAULT '',
    priority         TEXT NOT NULL DEFAULT 'normal',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tickets_queue ON tickets (queue);
CREATE INDEX IF NOT EXISTS idx_tickets_workflow ON tickets (workflow, workflow_version);

CREATE TABLE IF NOT EXISTS workflows (
    name    TEXT NOT NULL,
    version TEXT NOT NULL,
    def     JSONB NOT NULL,
    PRIMARY KEY (name, version)
);

-- Append-only event history. (run_id, seq) is the durable ordering key.
CREATE TABLE IF NOT EXISTS events (
    run_id      TEXT        NOT NULL,
    seq         BIGINT      NOT NULL,
    ticket_id   TEXT        NOT NULL,
    type        TEXT        NOT NULL,
    at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload     JSONB,
    schedule_at TIMESTAMPTZ, -- set for durable timers (delay steps, SLAs)
    PRIMARY KEY (run_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_events_ticket ON events (ticket_id);
CREATE INDEX IF NOT EXISTS idx_events_schedule ON events (schedule_at) WHERE schedule_at IS NOT NULL;
