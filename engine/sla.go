package engine

import (
	"context"
	"encoding/json"
)

// CheckSLA evaluates Service-Level-Agreement deadlines for an active run and
// escalates (flags + optionally jumps to an escalation step) when breached. It
// is idempotent: an SLA that has already breached is never re-flagged.
func (e *Engine) CheckSLA(ctx context.Context, run *WorkflowRun, def WorkflowDef) {
	if len(def.SLAs) == 0 {
		return
	}
	if run.Status != RunStatusActive && run.Status != RunStatusWaiting {
		return
	}
	events, err := e.log.History(ctx, run.ID)
	if err != nil {
		return
	}
	breached := map[string]bool{}
	for _, ev := range events {
		if ev.Type == EventSLABreached {
			var p SLABreachedPayload
			_ = json.Unmarshal(ev.Payload, &p)
			breached[p.Name] = true
		}
	}
	now := e.now()
	for _, sla := range def.SLAs {
		if breached[sla.Name] {
			continue
		}
		deadline := run.CreatedAt.Add(sla.Duration)
		if now.After(deadline) {
			_, _ = e.append(ctx, Event{TicketID: run.TicketID, RunID: run.ID, Type: EventSLABreached,
				Payload: mustJSON(SLABreachedPayload{Name: sla.Name, Step: run.Current})})
			if sla.OnBreach != "" {
				_ = e.jumpTo(ctx, run, def, sla.OnBreach)
			}
		}
	}
}

// jumpTo schedules a step out-of-band (used by SLA escalation). It avoids
// double-scheduling an already-active step.
func (e *Engine) jumpTo(ctx context.Context, run *WorkflowRun, def WorkflowDef, step string) error {
	if st, ok := run.Steps[step]; ok && (st.Status == StepPending || st.Status == StepRunning || st.Status == StepWaiting) {
		return nil
	}
	if _, err := e.scheduleStep(ctx, run.ID, run.TicketID, def, step); err != nil {
		return err
	}
	nr, err := e.GetRun(ctx, run.ID)
	if err == nil {
		e.dispatch(ctx, nr, def)
	}
	return nil
}
