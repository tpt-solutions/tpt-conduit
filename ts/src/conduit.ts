// TPT Conduit — TypeScript workflow DSL
//
// This builder produces exactly the same internal representation (IR) that the
// Go YAML DSL compiles to: `engine.WorkflowDef` (see engine/model.go). The
// engine consumes only that IR, so a workflow authored in TypeScript is
// indistinguishable at runtime from one authored in YAML.
//
// Durations in the IR are Go `time.Duration` values, i.e. integer nanoseconds.
// Use the `Duration` helper to express friendly units.

export type StepKind = "task" | "approval" | "delay";

export interface Approver {
  role: string;
  user?: string;
}

export interface StepDef {
  name: string;
  kind: StepKind;
  task?: string;
  approval?: { chain: Approver[] };
  delay?: { duration: number };
  next?: string;
  on_error?: string;
  assign_to?: string;
  retry?: { max_attempts?: number; delay?: number };
}

export interface SLADef {
  name: string;
  duration: number; // nanoseconds
  on_breach?: string;
}

export interface RoutingRule {
  if: Record<string, unknown>;
  queue: string;
  assignee?: string;
  priority?: string;
}

export interface WorkflowDef {
  name: string;
  version: string;
  description: string;
  initial: string;
  steps: StepDef[];
  slas?: SLADef[];
  routing?: RoutingRule[];
}

/** Duration helpers: the IR stores durations as integer nanoseconds. */
export const Duration = {
  nanoseconds: (n: number): number => n,
  microseconds: (n: number): number => n * 1_000,
  milliseconds: (n: number): number => n * 1_000_000,
  seconds: (n: number): number => n * 1_000_000_000,
  minutes: (n: number): number => n * 60 * 1_000_000_000,
  hours: (n: number): number => n * 3_600 * 1_000_000_000,
  days: (n: number): number => n * 86_400 * 1_000_000_000,
};

export interface StepOpts {
  next?: string;
  onError?: string;
  assignTo?: string;
  retries?: number;
  retryDelayNs?: number;
}

/** Define an automated/worker-executed task step. */
export function task(name: string, handler: string, opts: StepOpts = {}): StepDef {
  const s: StepDef = { name, kind: "task", task: handler };
  applyStepOpts(s, opts);
  return s;
}

/** Define a multi-step human approval chain. */
export function approval(name: string, chain: Approver[], opts: StepOpts = {}): StepDef {
  const s: StepDef = { name, kind: "approval", approval: { chain } };
  applyStepOpts(s, opts);
  return s;
}

/** Define a durable timer step. `durationNs` is nanoseconds. */
export function delay(name: string, durationNs: number, opts: StepOpts = {}): StepDef {
  const s: StepDef = { name, kind: "delay", delay: { duration: durationNs } };
  applyStepOpts(s, opts);
  return s;
}

function applyStepOpts(s: StepDef, opts: StepOpts): void {
  if (opts.next) s.next = opts.next;
  if (opts.onError) s.on_error = opts.onError;
  if (opts.assignTo) s.assign_to = opts.assignTo;
  if (opts.retries !== undefined || opts.retryDelayNs !== undefined) {
    s.retry = { max_attempts: opts.retries, delay: opts.retryDelayNs };
  }
}

export function sla(name: string, durationNs: number, onBreach?: string): SLADef {
  const s: SLADef = { name, duration: durationNs };
  if (onBreach) s.on_breach = onBreach;
  return s;
}

export function route(
  ifFields: Record<string, unknown>,
  target: { queue: string; assignee?: string; priority?: string },
): RoutingRule {
  return { if: ifFields, queue: target.queue, assignee: target.assignee, priority: target.priority };
}

/** Fluent builder for a workflow definition. */
export class Workflow {
  private def: WorkflowDef;

  constructor(meta: { name: string; version: string; description?: string; initial?: string }) {
    this.def = {
      name: meta.name,
      version: meta.version,
      description: meta.description ?? "",
      initial: meta.initial ?? "",
      steps: [],
      slas: [],
      routing: [],
    };
  }

  initial(name: string): this {
    this.def.initial = name;
    return this;
  }

  step(s: StepDef): this {
    this.def.steps.push(s);
    return this;
  }

  sla(s: SLADef): this {
    this.def.slas!.push(s);
    return this;
  }

  route(r: RoutingRule): this {
    this.def.routing!.push(r);
    return this;
  }

  /** Validate structural invariants (matching the engine's compile checks). */
  validate(): void {
    if (!this.def.name || !this.def.version) {
      throw new Error("workflow name and version are required");
    }
    if (this.def.steps.length === 0) {
      throw new Error("workflow must declare at least one step");
    }
    if (!this.def.initial) this.def.initial = this.def.steps[0].name;
    const names = new Set<string>();
    for (const s of this.def.steps) {
      if (names.has(s.name)) throw new Error(`duplicate step "${s.name}"`);
      names.add(s.name);
      if (s.kind === "task" && !s.task) throw new Error(`task step "${s.name}" requires a task handler`);
      if (s.kind === "approval" && (!s.approval || s.approval.chain.length === 0)) {
        throw new Error(`approval step "${s.name}" requires a non-empty chain`);
      }
      if (s.kind === "delay" && !s.delay) throw new Error(`delay step "${s.name}" requires a duration`);
    }
    for (const s of this.def.steps) {
      if (s.next && !names.has(s.next)) throw new Error(`step "${s.name}" next="${s.next}" not found`);
      if (s.on_error && !names.has(s.on_error)) throw new Error(`step "${s.name}" on_error="${s.on_error}" not found`);
    }
  }

  build(): WorkflowDef {
    this.validate();
    return JSON.parse(JSON.stringify(this.def)) as WorkflowDef;
  }

  toJSON(): string {
    return JSON.stringify(this.build(), null, 2);
  }
}
