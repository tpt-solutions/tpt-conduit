import { test } from "node:test";
import assert from "node:assert/strict";

import {
  Workflow,
  task,
  approval,
  delay,
  sla,
  route,
  Duration,
} from "../src/index.ts";

test("task step builds the engine IR shape", () => {
  const wf = new Workflow({ name: "w", version: "1.0.0" })
    .step(task("a", "do_a", { next: "b" }))
    .step(task("b", "do_b"))
    .build();
  assert.equal(wf.name, "w");
  assert.equal(wf.initial, "a");
  assert.equal(wf.steps[0].kind, "task");
  assert.equal(wf.steps[0].task, "do_a");
  assert.equal(wf.steps[0].next, "b");
  assert.equal(wf.steps[0].retry, undefined);
});

test("approval chain and SLA compile", () => {
  const wf = new Workflow({ name: "w", version: "1.0.0" })
    .step(
      approval("signoff", [{ role: "manager" }, { role: "director" }], { next: "done" }),
    )
    .step(task("done", "finish"))
    .sla(sla("breach", Duration.minutes(15), "signoff"))
    .build();
  assert.equal(wf.steps[0].approval!.chain.length, 2);
  assert.equal(wf.slas![0].duration, Duration.minutes(15));
  assert.equal(wf.slas![0].on_breach, "signoff");
});

test("routing rules compile", () => {
  const wf = new Workflow({ name: "w", version: "1.0.0" })
    .step(task("a", "do_a"))
    .route(route({ tier: "gold" }, { queue: "vip", priority: "high" }))
    .build();
  assert.equal(wf.routing![0].queue, "vip");
  assert.equal((wf.routing![0].if as Record<string, unknown>).tier, "gold");
});

test("delay durations are nanoseconds", () => {
  const wf = new Workflow({ name: "w", version: "1.0.0" })
    .step(delay("wait", Duration.hours(1)))
    .build();
  assert.equal(wf.steps[0].delay!.duration, 3_600_000_000_000);
});

test("validation rejects duplicate steps and dangling next", () => {
  assert.throws(() =>
    new Workflow({ name: "w", version: "1.0.0" })
      .step(task("a", "do_a", { next: "missing" }))
      .build(),
  );
  assert.throws(() =>
    new Workflow({ name: "w", version: "1.0.0" })
      .step(task("a", "do_a"))
      .step(task("a", "do_b"))
      .build(),
  );
});
