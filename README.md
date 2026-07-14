# TPT Conduit

**The "ServiceNow Disruptor" — an open, headless, API-first workflow & ITSM engine.**

Every Fortune 500 company relies on IT Service Management (ITSM) and workflow
engines for everything from IT helpdesk tickets to HR onboarding and asset
tracking. That space is dominated by ServiceNow: a proprietary black box that
costs millions to license, takes years to implement, and requires certified
developers just to change a dropdown. TPT Conduit reimagines ITSM as a modern,
programmable, free public good.

Instead of locking users into a rigid UI, Conduit provides a blazing-fast,
durable execution engine and treats **workflows as code**. Developers and IT
admins define approval chains, ticket routing, and SLA escalations in simple
**YAML** or **TypeScript**, version-controlled in Git. A beautiful, white-label
React frontend consumes a headless GraphQL/gRPC API.

Releasing TPT Conduit as an Apache 2.0 public good democratizes internal
tooling: companies build custom workflows in days, not months, and a community
of shared templates erodes the ServiceNow monopoly.

## Tech stack

| Layer            | Choice                                                            |
| ---------------- | ----------------------------------------------------------------- |
| Workflow engine  | **Go**, Temporal-inspired durable, fault-tolerant execution       |
| API layer        | **GraphQL** + **gRPC** for high-performance state queries/mutations |
| Workflow DSL     | **YAML** and **TypeScript**, compiled to one shared IR            |
| Frontend         | **Next.js** (React) with white-label theming and a plugin model   |
| Storage          | **PostgreSQL** first; CockroachDB/Cassandra considered for scale  |

## Project layout

```
/engine      Durable Go workflow engine (domain model, event log, worker, DSL IR)
  /dsl       YAML workflow DSL compiler -> shared IR
  /examples  Reference workflow templates (helpdesk, HR onboarding, asset tracking, approval chain)
/api         GraphQL + gRPC service surface (Phase 2)
/web         Next.js frontend (Phase 4)
/docs        Documentation site (Phase 6)
```

## Architecture principles

- **Event-sourced & durable.** Workflow state is never stored directly. Every
  effect is an append-only event; run state is always *derived* by replaying
  the log. A process can crash at any moment and resume exactly where it left
  off via `Engine.Recover`.
- **Generic / domain-agnostic.** A `Ticket` is just structured `Fields`; one
  engine serves helpdesk, HR, and asset tracking alike.
- **Primitive-based.** Approvals, timers, retries, SLA escalation, and routing
  are composable step kinds, not bespoke code paths.

## Quick start

```bash
# 1. Start Postgres (applies engine/schema.sql automatically)
docker compose up -d

# 2. Build & test the engine
go build ./...
go test ./...            # in-memory backend
PG_TEST=1 DATABASE_URL='postgres://conduit:conduit@localhost:5432/conduit?sslmode=disable' \
  go test ./engine -run Postgres
```

## Status

Phases 0–1 are implemented: repository scaffolding, the durable Go engine
(domain model, event-sourced log + replay, Postgres persistence, in-process
worker pool, retry/timer/SLA/approval/routing primitives) with crash-recovery
and replay tests. Phases 2+ (API surface, frontend, distributed scale-out) are
outlined in `TODO.md`.
