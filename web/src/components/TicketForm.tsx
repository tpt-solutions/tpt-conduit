"use client";

import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { graphqlRequest } from "@/lib/graphql-client";
import { CREATE_TICKET_MUTATION } from "@/lib/queries";
import { Workflow } from "@/lib/types";

export function TicketForm({ workflows }: { workflows: Workflow[] }) {
  const router = useRouter();
  const versionsByWorkflow = new Map<string, Workflow[]>();
  for (const w of workflows) {
    versionsByWorkflow.set(w.name, [...(versionsByWorkflow.get(w.name) || []), w]);
  }
  const workflowNames = [...versionsByWorkflow.keys()];

  const [workflow, setWorkflow] = useState(workflowNames[0] || "");
  const [version, setVersion] = useState(versionsByWorkflow.get(workflowNames[0] || "")?.[0]?.version || "");
  const [title, setTitle] = useState("");
  const [fieldsText, setFieldsText] = useState("{}");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const availableVersions = versionsByWorkflow.get(workflow) || [];

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);

    let fields: Record<string, unknown> = {};
    try {
      fields = fieldsText.trim() ? JSON.parse(fieldsText) : {};
    } catch {
      setError("Fields must be valid JSON");
      return;
    }

    setSubmitting(true);
    try {
      const data = await graphqlRequest<{ createTicket: { id: string } }>(CREATE_TICKET_MUTATION, {
        input: { workflow, version, title, fields },
      });
      router.push(`/tickets/${data.createTicket.id}`);
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create ticket");
    } finally {
      setSubmitting(false);
    }
  }

  if (workflowNames.length === 0) {
    return <div className="empty-state">No workflows are registered on the backend yet.</div>;
  }

  return (
    <form onSubmit={handleSubmit} className="card">
      {error ? <div className="error-banner">{error}</div> : null}
      <div className="field">
        <label htmlFor="title">Title</label>
        <input id="title" required value={title} onChange={(e) => setTitle(e.target.value)} />
      </div>
      <div className="field">
        <label htmlFor="workflow">Workflow</label>
        <select
          id="workflow"
          value={workflow}
          onChange={(e) => {
            const name = e.target.value;
            setWorkflow(name);
            setVersion(versionsByWorkflow.get(name)?.[0]?.version || "");
          }}
        >
          {workflowNames.map((name) => (
            <option key={name} value={name}>
              {name}
            </option>
          ))}
        </select>
      </div>
      <div className="field">
        <label htmlFor="version">Version</label>
        <select id="version" value={version} onChange={(e) => setVersion(e.target.value)}>
          {availableVersions.map((w) => (
            <option key={w.version} value={w.version}>
              {w.version}
            </option>
          ))}
        </select>
      </div>
      <div className="field">
        <label htmlFor="fields">Fields (JSON)</label>
        <textarea
          id="fields"
          rows={6}
          value={fieldsText}
          onChange={(e) => setFieldsText(e.target.value)}
          style={{ fontFamily: "ui-monospace, monospace" }}
        />
      </div>
      <button type="submit" className="btn btn-primary" disabled={submitting}>
        {submitting ? "Creating…" : "Create ticket"}
      </button>
    </form>
  );
}
