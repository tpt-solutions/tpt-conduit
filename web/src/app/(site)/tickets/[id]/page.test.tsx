import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/lib/graphql", () => ({
  graphqlFetch: vi.fn(),
}));
vi.mock("next/navigation", () => ({
  notFound: vi.fn(() => {
    throw new Error("NEXT_NOT_FOUND");
  }),
}));

import { graphqlFetch } from "@/lib/graphql";
import { notFound } from "next/navigation";
import TicketDetailPage from "./page";
import type { Ticket, WorkflowRun } from "@/lib/types";

const mockedFetch = vi.mocked(graphqlFetch);

const ticket: Ticket = {
  id: "ticket-1",
  workflow: "generic_approval_chain",
  workflowVersion: "1.0.0",
  title: "Purchase approval",
  fields: { amount: 500 },
  assignee: "boss@example.com",
  queue: "finance",
  priority: "medium",
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-02T00:00:00Z",
};

function makeRun(overrides: Partial<WorkflowRun> = {}): WorkflowRun {
  return {
    id: "run-1",
    ticketId: "ticket-1",
    workflow: "generic_approval_chain",
    workflowVersion: "1.0.0",
    status: "ACTIVE",
    currentStep: "review",
    steps: [],
    output: null,
    failed: null,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-02T00:00:00Z",
    ...overrides,
  };
}

describe("TicketDetailPage", () => {
  beforeEach(() => {
    mockedFetch.mockReset();
    vi.mocked(notFound).mockClear();
  });

  it("renders ticket fields and associated runs", async () => {
    mockedFetch.mockImplementation(async (query: string) => {
      if (query.includes("ticket(")) return { ticket };
      return { runs: [makeRun(), makeRun({ id: "run-2", ticketId: "other-ticket" })] };
    });

    render(await TicketDetailPage({ params: Promise.resolve({ id: "ticket-1" }) }));

    expect(screen.getByRole("heading", { name: "Purchase approval" })).toBeInTheDocument();
    expect(screen.getByText("generic_approval_chain@1.0.0")).toBeInTheDocument();
    expect(screen.getByText(/"amount": 500/)).toBeInTheDocument();

    // Only the run for this ticket should be listed, not the other ticket's run.
    expect(screen.getByRole("link", { name: "run-1" })).toHaveAttribute("href", "/runs/run-1");
    expect(screen.queryByRole("link", { name: "run-2" })).not.toBeInTheDocument();
  });

  it("shows an empty state when the ticket has no runs", async () => {
    mockedFetch.mockImplementation(async (query: string) => {
      if (query.includes("ticket(")) return { ticket };
      return { runs: [] };
    });

    render(await TicketDetailPage({ params: Promise.resolve({ id: "ticket-1" }) }));
    expect(screen.getByText(/no runs for this ticket/i)).toBeInTheDocument();
  });

  it("calls notFound when the ticket does not exist", async () => {
    mockedFetch.mockImplementation(async (query: string) => {
      if (query.includes("ticket(")) return { ticket: null };
      return { runs: [] };
    });

    await expect(TicketDetailPage({ params: Promise.resolve({ id: "missing" }) })).rejects.toThrow("NEXT_NOT_FOUND");
    expect(notFound).toHaveBeenCalled();
  });
});
