import Link from "next/link";
import { graphqlFetch } from "@/lib/graphql";
import { TICKETS_QUERY } from "@/lib/queries";
import { Ticket } from "@/lib/types";

export default async function TicketsPage() {
  const data = await graphqlFetch<{ tickets: Ticket[] }>(TICKETS_QUERY);
  const tickets = data.tickets;

  return (
    <>
      <div className="page-header">
        <h1>Tickets</h1>
        <Link href="/tickets/new" className="btn btn-primary">
          + New ticket
        </Link>
      </div>
      <div className="card" style={{ padding: 0 }}>
        {tickets.length === 0 ? (
          <div className="empty-state">No tickets yet.</div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Title</th>
                <th>Workflow</th>
                <th>Queue</th>
                <th>Priority</th>
                <th>Assignee</th>
                <th>Created</th>
              </tr>
            </thead>
            <tbody>
              {tickets.map((t) => (
                <tr key={t.id}>
                  <td>
                    <Link href={`/tickets/${t.id}`}>{t.title}</Link>
                  </td>
                  <td className="muted">
                    {t.workflow}@{t.workflowVersion}
                  </td>
                  <td className="muted">{t.queue || "—"}</td>
                  <td className="muted">{t.priority || "—"}</td>
                  <td className="muted">{t.assignee || "—"}</td>
                  <td className="muted">{new Date(t.createdAt).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </>
  );
}
