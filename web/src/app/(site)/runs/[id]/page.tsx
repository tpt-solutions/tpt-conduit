import Link from "next/link";
import { notFound } from "next/navigation";
import { graphqlFetch } from "@/lib/graphql";
import { RUN_QUERY, EVENTS_QUERY } from "@/lib/queries";
import { WorkflowRun, ConduitEvent } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";
import { ApprovalActions } from "@/components/ApprovalActions";
import { CancelRunButton } from "@/components/CancelRunButton";

const TERMINAL_STATUSES = new Set(["COMPLETED", "FAILED", "CANCELLED"]);

export default async function RunDetailPage({ params }: { params: { id: string } }) {
  const [runData, eventsData] = await Promise.all([
    graphqlFetch<{ run: WorkflowRun | null }>(RUN_QUERY, { id: params.id }),
    graphqlFetch<{ events: ConduitEvent[] }>(EVENTS_QUERY, { runId: params.id }),
  ]);

  if (!runData.run) {
    notFound();
  }
  const run = runData.run;
  const events = [...eventsData.events].sort((a, b) => b.seq - a.seq);

  return (
    <>
      <div className="page-header">
        <div>
          <h1>Run {run.id}</h1>
          <p className="muted" style={{ marginTop: 4 }}>
            Ticket <Link href={`/tickets/${run.ticketId}`}>{run.ticketId}</Link> · {run.workflow}@
            {run.workflowVersion}
          </p>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <StatusBadge status={run.status} />
          {!TERMINAL_STATUSES.has(run.status) && <CancelRunButton runId={run.id} />}
        </div>
      </div>

      {run.failed ? <div className="error-banner">Failed: {run.failed}</div> : null}

      <h2 style={{ fontSize: 15 }}>Steps</h2>
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

      <h2 style={{ fontSize: 15, marginTop: 24 }}>Event log</h2>
      <ul className="timeline card">
        {events.length === 0 ? (
          <li className="empty-state" style={{ width: "100%" }}>
            No events recorded.
          </li>
        ) : (
          events.map((e) => (
            <li key={e.seq}>
              <span className="dot" />
              <div style={{ flex: 1 }}>
                <div style={{ display: "flex", justifyContent: "space-between" }}>
                  <strong>{e.type}</strong>
                  <span className="muted" style={{ fontSize: 12 }}>
                    {new Date(e.at).toLocaleString()}
                  </span>
                </div>
                {e.payload ? (
                  <pre style={{ fontSize: 12, margin: "4px 0 0", overflowX: "auto" }}>
                    {JSON.stringify(e.payload, null, 2)}
                  </pre>
                ) : null}
              </div>
            </li>
          ))
        )}
      </ul>
    </>
  );
}
