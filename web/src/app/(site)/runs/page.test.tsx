import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/lib/graphql", () => ({
  graphqlFetch: vi.fn(),
}));

import { graphqlFetch } from "@/lib/graphql";
import RunsPage from "./page";
import type { WorkflowRun } from "@/lib/types";

const mockedFetch = vi.mocked(graphqlFetch);

function makeRun(overrides: Partial<WorkflowRun> = {}): WorkflowRun {
  return {
    id: "run-1",
    ticketId: "ticket-1",
    workflow: "it_helpdesk",
    workflowVersion: "1.0.0",
    status: "ACTIVE",
    currentStep: "triage",
    steps: [],
    output: null,
    failed: null,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("RunsPage", () => {
  beforeEach(() => {
    mockedFetch.mockReset();
  });

  it("shows an empty state when there are no runs", async () => {
    mockedFetch.mockResolvedValue({ runs: [] });
    render(await RunsPage());
    expect(screen.getByText(/no workflow runs yet/i)).toBeInTheDocument();
  });

  it("lists runs most-recently-updated first, linking to run and ticket", async () => {
    mockedFetch.mockResolvedValue({
      runs: [
        makeRun({ id: "run-old", updatedAt: "2026-01-01T00:00:00Z" }),
        makeRun({ id: "run-new", updatedAt: "2026-02-01T00:00:00Z" }),
      ],
    });

    render(await RunsPage());

    const runLinks = screen.getAllByRole("link", { name: /^run-/ });
    expect(runLinks.map((l) => l.textContent)).toEqual(["run-new", "run-old"]);
    expect(runLinks[0]).toHaveAttribute("href", "/runs/run-new");
    expect(screen.getAllByRole("link", { name: "ticket-1" })[0]).toHaveAttribute("href", "/tickets/ticket-1");
  });
});
