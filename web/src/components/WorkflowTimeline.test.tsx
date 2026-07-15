import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), refresh: vi.fn() }),
}));
vi.mock("@/lib/graphql-client", () => ({
  graphqlRequest: vi.fn().mockResolvedValue(undefined),
}));

import { WorkflowTimeline } from "@/components/WorkflowTimeline";
import type { WorkflowRun } from "@/lib/types";

function makeRun(steps: WorkflowRun["steps"], currentStep: string | null): WorkflowRun {
  return {
    id: "run-1",
    ticketId: "ticket-1",
    workflow: "it_helpdesk",
    workflowVersion: "1.0.0",
    status: "ACTIVE",
    currentStep,
    steps,
    output: null,
    failed: null,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  };
}

const triage = {
  name: "triage",
  kind: "task" as const,
  status: "COMPLETED" as const,
  attempt: 1,
  output: null,
  error: null,
  dueAt: null,
  approval: null,
};

const approvalPending = {
  name: "manager_approval",
  kind: "approval" as const,
  status: "PENDING" as const,
  attempt: 1,
  output: null,
  error: null,
  dueAt: null,
  approval: {
    chain: [{ role: "manager", user: "boss@example.com" }],
    index: 0,
    status: "pending" as const,
    decidedBy: null,
  },
};

const approvalGranted = {
  name: "manager_approval",
  kind: "approval" as const,
  status: "COMPLETED" as const,
  attempt: 1,
  output: null,
  error: null,
  dueAt: null,
  approval: {
    chain: [{ role: "manager", user: "boss@example.com" }],
    index: 0,
    status: "granted" as const,
    decidedBy: "boss@example.com",
  },
};

describe("WorkflowTimeline", () => {
  it("renders every step with its status and kind", () => {
    render(<WorkflowTimeline run={makeRun([triage, approvalPending], "manager_approval")} />);

    expect(screen.getByText("triage")).toBeInTheDocument();
    expect(screen.getByText("manager_approval")).toBeInTheDocument();
    expect(screen.getAllByText("task").length).toBeGreaterThan(0);
    expect(screen.getAllByText("approval").length).toBeGreaterThan(0);
  });

  it("marks the current step", () => {
    const { container } = render(
      <WorkflowTimeline run={makeRun([triage, approvalPending], "manager_approval")} />
    );
    const current = container.querySelector(".dot.current");
    expect(current).not.toBeNull();
  });

  it("shows the approval chain and required approver", () => {
    render(<WorkflowTimeline run={makeRun([approvalPending], "manager_approval")} />);
    expect(screen.getByText(/manager:boss@example.com/)).toBeInTheDocument();
  });

  it("renders approval actions only for a pending approval", () => {
    const { rerender } = render(<WorkflowTimeline run={makeRun([approvalPending], "manager_approval")} />);
    expect(screen.getByRole("button", { name: /approve/i })).toBeInTheDocument();

    rerender(<WorkflowTimeline run={makeRun([approvalGranted], "manager_approval")} />);
    expect(screen.queryByRole("button", { name: /approve/i })).not.toBeInTheDocument();
    expect(screen.getByText(/by boss@example.com/)).toBeInTheDocument();
  });

  it("renders an error banner for a failed step", () => {
    const failed = {
      ...triage,
      status: "FAILED" as const,
      error: "boom",
    };
    render(<WorkflowTimeline run={makeRun([failed], "triage")} />);
    expect(screen.getByText("boom")).toBeInTheDocument();
  });
});
