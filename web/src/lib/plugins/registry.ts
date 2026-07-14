// Plugin loading mechanism (TODO.md Phase 4 — "Decide plugin loading mechanism").
//
// Decision: build-time static registration. Plugins are normal modules that
// import this registry and register a `PluginModule` (typically from a central
// `plugins/index.ts` aggregator that statically imports each built-in plugin).
// No runtime `import()` of arbitrary URLs is used — that keeps the trust model
// simple (everything ships in the same bundle and is reviewed at build time)
// and works within Next.js' server/client component boundaries.
//
// The seam for a future runtime mechanism (sandboxed dynamic import, marketplace
// plugins) is intentionally narrow: swap the backing store of `registry` for a
// loader that fetches module manifests, and the `getX` resolvers below are
// unchanged. That change is deferred until there is a real external-plugin need.

import type { Ticket, WorkflowRun } from "@/lib/types";
import type { ConduitPlugin, PluginModule } from "./types";

const registry: PluginModule[] = [];

/** Register a plugin module. Safe to call multiple times; duplicates are ignored. */
export function registerPlugin(module: PluginModule): void {
  if (registry.some((m) => m.manifest.id === module.manifest.id)) return;
  registry.push(module);
}

/** All registered plugin modules, in registration order. */
export function getRegisteredPlugins(): readonly PluginModule[] {
  return registry;
}

/** Ticket-view plugins whose `matches()` predicate accepts the given ticket. */
export function getTicketViewPlugins(ticket: Ticket): ConduitPlugin[] {
  const views: ConduitPlugin[] = [];
  for (const m of registry) {
    for (const p of m.plugins) {
      if (p.kind === "ticket-view" && p.matches(ticket)) views.push(p);
    }
  }
  return views;
}

/** Dashboard-widget plugins (rendered on a future overview page). */
export function getDashboardWidgetPlugins(runs: WorkflowRun[] = []): ConduitPlugin[] {
  const widgets: ConduitPlugin[] = [];
  for (const m of registry) {
    for (const p of m.plugins) {
      if (p.kind === "dashboard-widget") widgets.push(p);
    }
  }
  return widgets;
}
