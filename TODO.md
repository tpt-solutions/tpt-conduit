# TPT Conduit — TODO

Checklist derived from `spec.txt`. Decisions locked in: Go engine, PostgreSQL first, full GraphQL+gRPC and YAML+TypeScript DSL surface, generic/domain-agnostic ticket model, single-tenant basic auth for now, bottom-up phasing.

## Phase 0 — Project setup
- [x] Repo scaffolding (Go module, monorepo layout: `/engine`, `/api`, `/dsl`, `/web`, `/docs`)
- [x] Apache 2.0 LICENSE file
- [x] Docker Compose for local Postgres
- [x] CI skeleton (lint/test/build)
- [x] README with project vision summary from spec.txt

## Phase 1 — Core durable workflow engine (Go)
- [x] Define core domain model: Ticket, Workflow, WorkflowRun, Step/Task, State transition, Event
- [x] Durable execution primitives: event sourcing / append-only history log, replay-based state recovery (Temporal-inspired)
- [x] Postgres schema + persistence layer for tickets and workflow state history
- [x] Task queue / worker execution loop (in-process to start, no distributed scheduler yet)
- [x] Retry policy, timers, and SLA escalation primitives
- [x] Approval-chain primitive (multi-step human-in-the-loop task)
- [x] Ticket routing primitive (rule-based assignment)
- [x] Unit + integration tests for engine core (crash/replay correctness)

## Phase 2 — API layer
- [x] GraphQL schema: queries/mutations for tickets, workflows, workflow runs, approvals
- [x] gRPC service definitions (proto) mirroring core mutations for high-perf/service-to-service use
- [x] Auth middleware: API key or basic username/password auth (single-tenant)
- [x] API-level tests (schema conformance, auth enforcement)

## Phase 3 — Workflow definition DSLs
- [x] YAML DSL: schema/spec for defining workflows (steps, transitions, approvals, SLAs) declaratively
- [x] YAML parser/validator + compiler to internal workflow definition format
- [x] TypeScript DSL: typed builder API producing the same internal workflow definition format
- [x] Shared internal workflow-definition IR so both DSLs compile to one thing
- [x] Git-friendly workflow versioning story (how workflow defs are loaded/deployed from repo)
- [x] Example workflow templates: IT helpdesk ticket, HR onboarding, asset tracking, generic approval chain

## Phase 4 — Frontend (Next.js)
- [x] Base Next.js app scaffold, GraphQL client wiring
- [x] Core ticket views: list, detail, create/update
- [x] Workflow run visualization (current state, history/timeline)
- [x] Approval action UI (approve/reject a pending step)
- [x] White-label theming support (colors/logo config)
- [x] Plugin architecture (high-level placeholder only for now):
  - [x] Design plugin API surface (what a plugin can register: custom ticket views, dashboard widgets)
  - [ ] Decide plugin loading mechanism (revisit once core UI exists)
- [x] Basic auth UI (login)

## Phase 5 — Distributed scale-out (deferred until needed)
- [ ] Evaluate CockroachDB vs Cassandra for workflow state history at scale
- [ ] Migration path from single-node Postgres to distributed store
- [ ] Distributed task scheduling/worker coordination (multi-node execution engine)
- [ ] Multi-tenancy support
- [ ] SSO (SAML/OIDC) support
- [ ] Cloud deployment (Kubernetes manifests / Helm chart)

## Cross-cutting — Full test coverage
- [x] Engine: unit tests for every state transition, retry/timeout/SLA escalation path, and approval-chain branch
- [x] Engine: crash/replay correctness tests for all event-sourced code paths (not just happy path)
- [x] Persistence layer: integration tests against real Postgres for every query/migration
- [x] DSL: parser/validator/compiler tests for valid, invalid, and edge-case YAML and TypeScript workflow defs
- [x] API: GraphQL resolver tests (queries/mutations, error cases, auth-denied cases)
- [x] API: gRPC service tests mirroring GraphQL coverage
- [x] Auth middleware: tests for valid/invalid/missing credentials, API key and basic-auth paths
- [ ] Frontend: component tests for ticket views, workflow visualization, approval actions
- [ ] Frontend: end-to-end tests for core user flows (create ticket, run workflow, approve/reject step)
- [x] Coverage reporting wired into CI with an enforced minimum threshold (Go: 55% minimum; profile uploaded as artifact)
- [ ] Coverage badge/report published alongside CI results (badge automation pending)
