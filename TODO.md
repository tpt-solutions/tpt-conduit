# TPT Conduit — TODO

Checklist derived from `spec.txt`. Decisions locked in: Go engine, PostgreSQL first, full GraphQL+gRPC and YAML+TypeScript DSL surface, generic/domain-agnostic ticket model, single-tenant basic auth for now, bottom-up phasing.

## Phase 0 — Project setup
- [ ] Repo scaffolding (Go module, monorepo layout: `/engine`, `/api`, `/dsl`, `/web`, `/docs`)
- [ ] Apache 2.0 LICENSE file
- [ ] Docker Compose for local Postgres
- [ ] CI skeleton (lint/test/build)
- [ ] README with project vision summary from spec.txt

## Phase 1 — Core durable workflow engine (Go)
- [ ] Define core domain model: Ticket, Workflow, WorkflowRun, Step/Task, State transition, Event
- [ ] Durable execution primitives: event sourcing / append-only history log, replay-based state recovery (Temporal-inspired)
- [ ] Postgres schema + persistence layer for tickets and workflow state history
- [ ] Task queue / worker execution loop (in-process to start, no distributed scheduler yet)
- [ ] Retry policy, timers, and SLA escalation primitives
- [ ] Approval-chain primitive (multi-step human-in-the-loop task)
- [ ] Ticket routing primitive (rule-based assignment)
- [ ] Unit + integration tests for engine core (crash/replay correctness)

## Phase 2 — API layer
- [ ] GraphQL schema: queries/mutations for tickets, workflows, workflow runs, approvals
- [ ] gRPC service definitions (proto) mirroring core mutations for high-perf/service-to-service use
- [ ] Auth middleware: API key or basic username/password auth (single-tenant)
- [ ] API-level tests (schema conformance, auth enforcement)

## Phase 3 — Workflow definition DSLs
- [ ] YAML DSL: schema/spec for defining workflows (steps, transitions, approvals, SLAs) declaratively
- [ ] YAML parser/validator + compiler to internal workflow definition format
- [ ] TypeScript DSL: typed builder API producing the same internal workflow definition format
- [ ] Shared internal workflow-definition IR so both DSLs compile to one thing
- [ ] Git-friendly workflow versioning story (how workflow defs are loaded/deployed from repo)
- [ ] Example workflow templates: IT helpdesk ticket, HR onboarding, asset tracking, generic approval chain

## Phase 4 — Frontend (Next.js)
- [ ] Base Next.js app scaffold, GraphQL client wiring
- [ ] Core ticket views: list, detail, create/update
- [ ] Workflow run visualization (current state, history/timeline)
- [ ] Approval action UI (approve/reject a pending step)
- [ ] White-label theming support (colors/logo config)
- [ ] Plugin architecture (high-level placeholder only for now):
  - [ ] Design plugin API surface (what a plugin can register: custom ticket views, dashboard widgets)
  - [ ] Decide plugin loading mechanism (revisit once core UI exists)
- [ ] Basic auth UI (login)

## Phase 5 — Distributed scale-out (deferred until needed)
- [ ] Evaluate CockroachDB vs Cassandra for workflow state history at scale
- [ ] Migration path from single-node Postgres to distributed store
- [ ] Distributed task scheduling/worker coordination (multi-node execution engine)
- [ ] Multi-tenancy support
- [ ] SSO (SAML/OIDC) support
- [ ] Cloud deployment (Kubernetes manifests / Helm chart)

## Phase 6 — Community/ecosystem (longer-term)
- [ ] Public docs site
- [ ] Workflow template marketplace/sharing mechanism
- [ ] Contribution guidelines
