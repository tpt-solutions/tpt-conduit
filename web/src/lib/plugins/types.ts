// Placeholder plugin API surface (TODO.md Phase 4). Nothing in the app loads
// or executes these yet — this file only fixes the shape we intend plugins
// to conform to, so the real loader (registration, sandboxing, versioning)
// can be designed against a stable contract later.

import type { Ticket, WorkflowRun } from "@/lib/types";

export interface PluginManifest {
  id: string;
  name: string;
  version: string;
}

/** Registers an alternate/additional detail view for a ticket, e.g. a custom summary panel. */
export interface TicketViewPlugin {
  kind: "ticket-view";
  /** Return true if this plugin should render for the given ticket (e.g. by workflow name). */
  matches(ticket: Ticket): boolean;
  render(ticket: Ticket): unknown; // React node — left untyped until a loading mechanism exists
}

/** Registers a widget shown on a future dashboard/overview page. */
export interface DashboardWidgetPlugin {
  kind: "dashboard-widget";
  title: string;
  render(context: { runs: WorkflowRun[] }): unknown;
}

export type ConduitPlugin = TicketViewPlugin | DashboardWidgetPlugin;

export interface PluginModule {
  manifest: PluginManifest;
  plugins: ConduitPlugin[];
}
