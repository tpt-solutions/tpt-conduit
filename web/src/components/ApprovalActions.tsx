"use client";

import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { graphqlRequest } from "@/lib/graphql-client";
import { APPROVE_MUTATION, REJECT_MUTATION } from "@/lib/queries";

export function ApprovalActions({ runId, step }: { runId: string; step: string }) {
  const router = useRouter();
  const [by, setBy] = useState("");
  const [comment, setComment] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState<"approve" | "reject" | null>(null);

  async function handleDecision(e: FormEvent, decision: "approve" | "reject") {
    e.preventDefault();
    if (!by.trim()) {
      setError("Enter who is deciding");
      return;
    }
    setError(null);
    setSubmitting(decision);
    try {
      if (decision === "approve") {
        await graphqlRequest(APPROVE_MUTATION, { runId, step, by, comment: comment || null });
      } else {
        await graphqlRequest(REJECT_MUTATION, { runId, step, by, reason: comment || null });
      }
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Action failed");
    } finally {
      setSubmitting(null);
    }
  }

  return (
    <form className="card" style={{ marginTop: 8, background: "transparent" }}>
      {error ? <div className="error-banner">{error}</div> : null}
      <div className="field">
        <label htmlFor={`by-${step}`}>Decided by</label>
        <input id={`by-${step}`} value={by} onChange={(e) => setBy(e.target.value)} placeholder="you@example.com" />
      </div>
      <div className="field">
        <label htmlFor={`comment-${step}`}>Comment / reason (optional)</label>
        <input id={`comment-${step}`} value={comment} onChange={(e) => setComment(e.target.value)} />
      </div>
      <div style={{ display: "flex", gap: 10 }}>
        <button
          type="button"
          className="btn btn-primary"
          disabled={submitting !== null}
          onClick={(e) => handleDecision(e, "approve")}
        >
          {submitting === "approve" ? "Approving…" : "Approve"}
        </button>
        <button
          type="button"
          className="btn btn-danger"
          disabled={submitting !== null}
          onClick={(e) => handleDecision(e, "reject")}
        >
          {submitting === "reject" ? "Rejecting…" : "Reject"}
        </button>
      </div>
    </form>
  );
}
