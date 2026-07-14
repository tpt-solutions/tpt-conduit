// Central plugin aggregator (build-time static registration).
//
// Every built-in plugin is statically imported and registered here so it is
// bundled and available at runtime. To add a built-in plugin, import its module
// and call `registerPlugin` (or have the module self-register on import). This
// is the single registration point that the build-time loading mechanism relies
// on; see `registry.ts` for the rationale and the future runtime-loader seam.
import { registerPlugin } from "./registry";

export { registerPlugin, getRegisteredPlugins, getTicketViewPlugins, getDashboardWidgetPlugins } from "./registry";
export type { ConduitPlugin, PluginModule, TicketViewPlugin, DashboardWidgetPlugin, PluginManifest } from "./types";

// No built-in plugins ship yet. Register them here once a host page (e.g. a
// dashboard/overview) exists to render dashboard widgets, or once a workflow
// needs a custom ticket-detail view.
void registerPlugin;
