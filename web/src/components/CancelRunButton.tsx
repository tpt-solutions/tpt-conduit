"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { graphqlRequest } from "@/lib/graphql-client";
import { CANCEL_MUTATION } from "@/lib/queries";

export function CancelRunButton({ runId }: { runId: string }) {
  const router = useRouter();
  const [submitting, setSubmitting] = useState(false);

  async function handleCancel() {
    if (!window.confirm("Cancel this workflow run?")) return;
    setSubmitting(true);
    try {
      await graphqlRequest(CANCEL_MUTATION, { runId, reason: "Cancelled from console" });
      router.refresh();
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <button className="btn" onClick={handleCancel} disabled={submitting}>
      {submitting ? "Cancelling…" : "Cancel run"}
    </button>
  );
}
