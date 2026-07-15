import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/lib/graphql", () => ({
  graphqlFetch: vi.fn(),
}));
vi.mock("next/navigation", () => ({
  notFound: vi.fn(() => {
    throw new Error("NEXT_NOT_FOUND");
  }),
  useRouter: () => ({ push: vi.fn(), refresh: vi.fn() }),
}));

import { graphqlFetch } from "@/lib/graphql";
import { notFound } from "next/navigation";
import RunDetailPage from "./page";
import type { WorkflowRun, ConduitEvent } from "@/lib/types";

const mockedFetch = vi.mocked(graphqlFetch);

function makeRun(overrides: Partial<WorkflowRun> = {}): WorkflowRun {
  return {
    id: "run-1",
    ticketId: "ticket-1",
    workflow: "generic_approval_chain",
    workflowVersion: "1.0.0",
    status: "ACTIVE",
    currentStep: "review",
    steps: [
      {
        name: "request",
        kind: "task",
        status: "COMPLETED",
        attempt: 1,
        output: null,
        error: null,
        dueAt: null,
        approval: null,
      },
    ],
    output: null,
    failed: null,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-02T00:00:00Z",
    ...overrides,
  };
}

const events: ConduitEvent[] = [
  { seq: 1, type: "RunStarted", at: "2026-01-01T00:00:00Z", payload: null, scheduleAt: null },
  { seq: 2, type: "StepCompleted", at: "2026-01-01T01:00:00Z", payload: { step: "request" }, scheduleAt: null },
];

describe("RunDetailPage", () => {
  beforeEach(() => {
    mockedFetch.mockReset();
    vi.mocked(notFound).mockClear();
  });

  it("renders run header, timeline, and events newest-first", async () => {
    mockedFetch.mockImplementation(async (query: string) => {
      if (query.includes("run(")) return { run: makeRun() };
      return { events };
    });

    render(await RunDetailPage({ params: Promise.resolve({ id: "run-1" }) }));

    expect(screen.getByRole("heading", { name: "Run run-1" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "ticket-1" })).toHaveAttribute("href", "/tickets/ticket-1");
    expect(screen.getByRole("button", { name: /cancel run/i })).toBeInTheDocument();

    const eventTypes = screen.getAllByText(/RunStarted|StepCompleted/).map((el) => el.textContent);
    expect(eventTypes).toEqual(["StepCompleted", "RunStarted"]);
  });

  it("hides the cancel button and shows the failure banner for a terminal, failed run", async () => {
    mockedFetch.mockImplementation(async (query: string) => {
      if (query.includes("run(")) return { run: makeRun({ status: "FAILED", failed: "resolve: boom" }) };
      return { events: [] };
    });

    render(await RunDetailPage({ params: Promise.resolve({ id: "run-1" }) }));

    expect(screen.queryByRole("button", { name: /cancel run/i })).not.toBeInTheDocument();
    expect(screen.getByText(/failed: resolve: boom/i)).toBeInTheDocument();
    expect(screen.getByText(/no events recorded/i)).toBeInTheDocument();
  });

  it("calls notFound when the run does not exist", async () => {
    mockedFetch.mockImplementation(async (query: string) => {
      if (query.includes("run(")) return { run: null };
      return { events: [] };
    });

    await expect(RunDetailPage({ params: Promise.resolve({ id: "missing" }) })).rejects.toThrow("NEXT_NOT_FOUND");
    expect(notFound).toHaveBeenCalled();
  });
});
