import { writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

import { Workflow, task, approval, delay, sla, route, Duration } from "../src/index.ts";

// IT helpdesk ticket workflow, authored in the TypeScript DSL. It compiles to
// the exact same engine.WorkflowDef IR that the YAML DSL produces.
const helpdesk = new Workflow({
  name: "helpdesk",
  version: "1.0.0",
  description: "IT helpdesk ticket triage, assignment and resolution.",
})
  .initial("triage")
  .route(route({ category: "hardware" }, { queue: "hw-team", priority: "high" }))
  .route(route({ category: "access" }, { queue: "iam-team" }))
  .step(
    task("triage", "triage_ticket", {
      next: "resolve",
      onError: "escalate",
    }),
  )
  .step(
    approval("escalate", [
      { role: "team_lead" },
      { role: "manager" },
    ], { next: "resolve" }),
  )
  .step(
    task("resolve", "resolve_ticket", {
      next: "close",
    }),
  )
  .step(delay("close", Duration.hours(24), { next: "" }))
  .sla(sla("first_response", Duration.minutes(30), "escalate"))
  .build();

const here = dirname(fileURLToPath(import.meta.url));
writeFileSync(join(here, "helpdesk.json"), JSON.stringify(helpdesk, null, 2) + "\n");
console.log("wrote helpdesk.json");
console.log(JSON.stringify(helpdesk, null, 2));
