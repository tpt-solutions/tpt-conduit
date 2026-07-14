import { describe, it, expect } from "vitest";
import {
  registerPlugin,
  getRegisteredPlugins,
  getTicketViewPlugins,
  getDashboardWidgetPlugins,
} from "@/lib/plugins/registry";
import type { PluginModule } from "@/lib/plugins/types";
import type { Ticket } from "@/lib/types";

const helpdeskTicket: Ticket = {
  id: "t1",
  workflow: "it_helpdesk",
  workflowVersion: "1.0.0",
  title: "Broken laptop",
  fields: {},
  assignee: null,
  queue: null,
  priority: null,
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
};

const otherTicket: Ticket = { ...helpdeskTicket, workflow: "hr_onboarding" };

const helpdeskView: PluginModule = {
  manifest: { id: "helpdesk-view", name: "Helpdesk view", version: "1.0.0" },
  plugins: [
    {
      kind: "ticket-view",
      matches: (t) => t.workflow === "it_helpdesk",
      render: () => null,
    },
  ],
};

const overviewWidget: PluginModule = {
  manifest: { id: "overview-widget", name: "Overview widget", version: "1.0.0" },
  plugins: [{ kind: "dashboard-widget", title: "Open runs", render: () => null }],
};

describe("plugin registry", () => {
  it("registers a plugin and ignores duplicate ids", () => {
    registerPlugin(helpdeskView);
    registerPlugin(helpdeskView);
    const ids = getRegisteredPlugins().map((m) => m.manifest.id);
    expect(ids.filter((id) => id === "helpdesk-view")).toHaveLength(1);
  });

  it("returns ticket-view plugins matching the ticket", () => {
    registerPlugin(helpdeskView);
    const views = getTicketViewPlugins(helpdeskTicket);
    expect(views).toHaveLength(1);
    expect(views[0].kind).toBe("ticket-view");

    expect(getTicketViewPlugins(otherTicket)).toHaveLength(0);
  });

  it("returns all dashboard-widget plugins regardless of ticket", () => {
    registerPlugin(overviewWidget);
    const widgets = getDashboardWidgetPlugins([]);
    expect(widgets).toHaveLength(1);
    expect(widgets[0].kind).toBe("dashboard-widget");
  });
});
