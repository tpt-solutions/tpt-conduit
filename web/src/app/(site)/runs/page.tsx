import Link from "next/link";
import { graphqlFetch } from "@/lib/graphql";
import { RUNS_QUERY } from "@/lib/queries";
import { WorkflowRun } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";

export default async function RunsPage() {
  const data = await graphqlFetch<{ runs: WorkflowRun[] }>(RUNS_QUERY);
  const runs = [...data.runs].sort(
    (a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
  );

  return (
    <>
      <div className="page-header">
        <h1>Workflow runs</h1>
      </div>
      <div className="card" style={{ padding: 0 }}>
        {runs.length === 0 ? (
          <div className="empty-state">No workflow runs yet.</div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Run</th>
                <th>Workflow</th>
                <th>Status</th>
                <th>Current step</th>
                <th>Ticket</th>
                <th>Updated</th>
              </tr>
            </thead>
            <tbody>
              {runs.map((r) => (
                <tr key={r.id}>
                  <td>
                    <Link href={`/runs/${r.id}`}>{r.id}</Link>
                  </td>
                  <td className="muted">
                    {r.workflow}@{r.workflowVersion}
                  </td>
                  <td>
                    <StatusBadge status={r.status} />
                  </td>
                  <td className="muted">{r.currentStep || "—"}</td>
                  <td>
                    <Link href={`/tickets/${r.ticketId}`}>{r.ticketId}</Link>
                  </td>
                  <td className="muted">{new Date(r.updatedAt).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </>
  );
}
