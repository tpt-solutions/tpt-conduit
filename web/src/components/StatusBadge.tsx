const TONE_BY_STATUS: Record<string, string> = {
  ACTIVE: "badge-active",
  RUNNING: "badge-active",
  WAITING: "badge-warning",
  PENDING: "badge-neutral",
  COMPLETED: "badge-success",
  GRANTED: "badge-success",
  FAILED: "badge-danger",
  REJECTED: "badge-danger",
  CANCELLED: "badge-neutral",
  SKIPPED: "badge-neutral",
};

export function StatusBadge({ status }: { status: string }) {
  const tone = TONE_BY_STATUS[status.toUpperCase()] || "badge-neutral";
  return <span className={`badge ${tone}`}>{status}</span>;
}
