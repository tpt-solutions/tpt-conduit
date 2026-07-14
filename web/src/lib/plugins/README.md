# Plugin architecture (placeholder)

Design-only for now — see `types.ts` for the current API surface sketch. No
loader exists yet; nothing in the app imports these types outside this
folder.

## What a plugin can register (proposed)

- **Ticket view**: an alternate/supplementary detail panel for tickets, scoped
  by a `matches(ticket)` predicate (e.g. per-workflow custom views).
- **Dashboard widget**: a titled widget rendered on a future dashboard/overview
  page, given the current set of workflow runs.

## Loading mechanism (undecided — revisit once a dashboard page exists)

Candidates, not yet chosen:
- Build-time static import list (simplest, no dynamic loading, plugins ship in
  the same bundle).
- Runtime dynamic `import()` from a configured URL/package name (more
  flexible, but needs a sandboxing/trust story before it's safe).

No decision is needed until there's a second real consumer of the plugin
surface (i.e. once the dashboard page in Phase 4/5 exists to host widgets).
