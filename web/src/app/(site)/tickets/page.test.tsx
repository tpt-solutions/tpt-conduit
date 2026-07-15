import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/lib/graphql", () => ({
  graphqlFetch: vi.fn(),
}));

import { graphqlFetch } from "@/lib/graphql";
import TicketsPage from "./page";
import type { Ticket } from "@/lib/types";

const mockedFetch = vi.mocked(graphqlFetch);

function makeTicket(overrides: Partial<Ticket> = {}): Ticket {
  return {
    id: "ticket-1",
    workflow: "it_helpdesk",
    workflowVersion: "1.0.0",
    title: "Broken laptop",
    fields: {},
    assignee: null,
    queue: "it-desk",
    priority: "high",
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("TicketsPage", () => {
  beforeEach(() => {
    mockedFetch.mockReset();
  });

  it("shows an empty state when there are no tickets", async () => {
    mockedFetch.mockResolvedValue({ tickets: [] });
    render(await TicketsPage());
    expect(screen.getByText(/no tickets yet/i)).toBeInTheDocument();
  });

  it("lists tickets with links to their detail pages", async () => {
    mockedFetch.mockResolvedValue({ tickets: [makeTicket(), makeTicket({ id: "ticket-2", title: "VPN access", assignee: "boss@example.com" })] });
    render(await TicketsPage());

    const link = screen.getByRole("link", { name: "Broken laptop" });
    expect(link).toHaveAttribute("href", "/tickets/ticket-1");
    expect(screen.getByRole("link", { name: "VPN access" })).toHaveAttribute("href", "/tickets/ticket-2");
    expect(screen.getByText("boss@example.com")).toBeInTheDocument();
    expect(screen.getAllByText("—").length).toBeGreaterThan(0);
  });
});
