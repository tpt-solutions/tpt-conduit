# Plugin architecture

The plugin API surface is defined in `types.ts`. The loading mechanism is now
decided — see `registry.ts` for the implemented loader and `index.ts` for the
single build-time registration point.

## Loading mechanism — decided (build-time static registration)

Plugins are normal modules that register a `PluginModule` through
`registerPlugin` (typically aggregated from `index.ts`, which statically imports
every built-in plugin). No runtime `import()` of arbitrary URLs is used.

Why this mechanism:

- **Simple trust model.** Everything ships in the same bundle and is reviewed at
  build time. There is no untrusted code executing at runtime, so no sandboxing
  story is needed yet.
- **Works within Next.js.** Server and client components both import the registry
  directly; there is no dynamic-code-loading boundary to fight.
- **Narrow future seam.** If external/marketplace plugins are later required, the
  only change is swapping the backing store of `registry` for a loader that
  fetches and (sandboxed) evaluates module manifests. The `getTicketViewPlugins`
  / `getDashboardWidgetPlugins` resolvers stay unchanged.

## What a plugin can register

- **Ticket view**: an alternate/supplementary detail panel for tickets, scoped by
  a `matches(ticket)` predicate (e.g. per-workflow custom views).
- **Dashboard widget**: a titled widget rendered on a future dashboard/overview
  page, given the current set of workflow runs.

## Status

- API surface: designed (`types.ts`).
- Loader: implemented (`registry.ts` + `index.ts`), with unit tests.
- Host pages: not yet present. `getTicketViewPlugins` is ready to be consumed by
  the ticket-detail page, and `getDashboardWidgetPlugins` by a future dashboard.
  No built-in plugins are registered yet.
