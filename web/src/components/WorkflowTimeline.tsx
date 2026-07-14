import { StatusBadge } from "@/components/StatusBadge";
import { ApprovalActions } from "@/components/ApprovalActions";
import type { WorkflowRun } from "@/lib/types";

const TERMINAL_STATUSES = new Set(["COMPLETED", "FAILED", "CANCELLED"]);

/** Renders the step-by-step timeline for a workflow run (current state + history). */
export function WorkflowTimeline({ run }: { run: WorkflowRun }) {
  return (
    <ul className="timeline card">
      {run.steps.map((step) => {
        const isCurrent = step.name === run.currentStep;
        const needsDecision =
          step.kind === "approval" && step.approval && step.approval.status === "pending";
        return (
          <li key={step.name}>
            <span className={`dot ${isCurrent ? "current" : ""}`} />
            <div style={{ flex: 1 }}>
              <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                <strong>{step.name}</strong>
                <span className="muted" style={{ fontSize: 12 }}>
                  {step.kind}
                </span>
                <StatusBadge status={step.status} />
                {step.attempt > 1 && (
                  <span className="muted" style={{ fontSize: 12 }}>
                    attempt {step.attempt}
                  </span>
                )}
              </div>
              {step.error ? <div className="error-banner">{step.error}</div> : null}
              {step.approval ? (
                <div className="muted" style={{ fontSize: 13, marginTop: 4 }}>
                  Approval: <StatusBadge status={step.approval.status} />
                  {step.approval.decidedBy ? ` by ${step.approval.decidedBy}` : ""} — chain:{" "}
                  {step.approval.chain.map((a) => `${a.role}:${a.user}`).join(", ")}
                </div>
              ) : null}
              {needsDecision ? <ApprovalActions runId={run.id} step={step.name} /> : null}
            </div>
          </li>
        );
      })}
    </ul>
  );
}

export { TERMINAL_STATUSES };
