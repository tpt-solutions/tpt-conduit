import Link from "next/link";
import { notFound } from "next/navigation";
import { graphqlFetch } from "@/lib/graphql";
import { TICKET_QUERY, RUNS_QUERY } from "@/lib/queries";
import { Ticket, WorkflowRun } from "@/lib/types";
import { StatusBadge } from "@/components/StatusBadge";

export default async function TicketDetailPage({ params }: { params: { id: string } }) {
  const [ticketData, runsData] = await Promise.all([
    graphqlFetch<{ ticket: Ticket | null }>(TICKET_QUERY, { id: params.id }),
    graphqlFetch<{ runs: WorkflowRun[] }>(RUNS_QUERY),
  ]);

  if (!ticketData.ticket) {
    notFound();
  }
  const ticket = ticketData.ticket;
  const runs = runsData.runs.filter((r) => r.ticketId === ticket.id);

  return (
    <>
      <div className="page-header">
        <h1>{ticket.title}</h1>
      </div>

      <div className="card" style={{ marginBottom: 20 }}>
        <table>
          <tbody>
            <tr>
              <th style={{ width: 160 }}>Workflow</th>
              <td>
                {ticket.workflow}@{ticket.workflowVersion}
              </td>
            </tr>
            <tr>
              <th>Queue</th>
              <td>{ticket.queue || "—"}</td>
            </tr>
            <tr>
              <th>Priority</th>
              <td>{ticket.priority || "—"}</td>
            </tr>
            <tr>
              <th>Assignee</th>
              <td>{ticket.assignee || "—"}</td>
            </tr>
            <tr>
              <th>Created</th>
              <td>{new Date(ticket.createdAt).toLocaleString()}</td>
            </tr>
            <tr>
              <th>Updated</th>
              <td>{new Date(ticket.updatedAt).toLocaleString()}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <h2 style={{ fontSize: 15 }}>Fields</h2>
      <pre className="card" style={{ overflowX: "auto", fontSize: 13 }}>
        {JSON.stringify(ticket.fields, null, 2)}
      </pre>

      <h2 style={{ fontSize: 15 }}>Workflow runs</h2>
      <div className="card" style={{ padding: 0, marginBottom: 12 }}>
        {runs.length === 0 ? (
          <div className="empty-state">No runs for this ticket.</div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Run</th>
                <th>Status</th>
                <th>Current step</th>
                <th>Updated</th>
              </tr>
            </thead>
            <tbody>
              {runs.map((r) => (
                <tr key={r.id}>
                  <td>
                    <Link href={`/runs/${r.id}`}>{r.id}</Link>
                  </td>
                  <td>
                    <StatusBadge status={r.status} />
                  </td>
                  <td className="muted">{r.currentStep || "—"}</td>
                  <td className="muted">{new Date(r.updatedAt).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      <p className="muted" style={{ fontSize: 13 }}>
        Ticket editing isn&apos;t available yet — the backend GraphQL API only exposes{" "}
        <code>createTicket</code>, not an update mutation.
      </p>
    </>
  );
}
