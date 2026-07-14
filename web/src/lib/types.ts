export type RunStatus = "ACTIVE" | "WAITING" | "COMPLETED" | "FAILED" | "CANCELLED";
export type StepStatus = "PENDING" | "RUNNING" | "COMPLETED" | "FAILED" | "WAITING" | "SKIPPED";
export type StepKind = "task" | "approval" | "delay";
export type ApprovalStatus = "pending" | "granted" | "rejected";

export interface Approver {
  role: string;
  user: string;
}

export interface ApprovalDef {
  chain: Approver[];
}

export interface Step {
  name: string;
  kind: StepKind;
  task: string | null;
  next: string | null;
  onError: string | null;
  assignTo: string | null;
  approval: ApprovalDef | null;
  delay: unknown;
  retry: unknown;
}

export interface Workflow {
  name: string;
  version: string;
  description: string | null;
  initial: string;
  steps: Step[];
}

export interface Ticket {
  id: string;
  workflow: string;
  workflowVersion: string;
  title: string;
  fields: Record<string, unknown>;
  assignee: string | null;
  queue: string | null;
  priority: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface ApprovalState {
  chain: Approver[];
  index: number;
  status: ApprovalStatus;
  decidedBy: string | null;
}

export interface StepState {
  name: string;
  kind: StepKind;
  status: StepStatus;
  attempt: number;
  output: unknown;
  error: string | null;
  dueAt: string | null;
  approval: ApprovalState | null;
}

export interface WorkflowRun {
  id: string;
  ticketId: string;
  workflow: string;
  workflowVersion: string;
  status: RunStatus;
  currentStep: string | null;
  steps: StepState[];
  output: unknown;
  failed: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface ConduitEvent {
  seq: number;
  type: string;
  at: string;
  payload: unknown;
  scheduleAt: string | null;
}

export interface CreateTicketInput {
  workflow: string;
  version: string;
  title: string;
  fields?: Record<string, unknown>;
}
