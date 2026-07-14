package engine

// Router implements rule-based ticket assignment (routing primitive). Rules
// are evaluated in order; the first match sets queue/assignee/priority. This
// is intentionally simple and deterministic so that replays route identically.
type Router struct{}

// NewRouter returns an empty router.
func NewRouter() *Router { return &Router{} }

// Apply mutates the ticket's queue/assignee/priority based on the first
// matching routing rule. Unmatched fields are left unchanged.
func (r *Router) Apply(t *Ticket, rules []RoutingRule) {
	for i, rule := range rules {
		if matchRule(t.Fields, rule.If) {
			if rule.Queue != "" {
				t.Queue = rule.Queue
			}
			if rule.Assignee != "" {
				t.Assignee = rule.Assignee
			}
			if rule.Priority != "" {
				t.Priority = rule.Priority
			}
			// Record rule index via a synthetic field for observability.
			if t.Fields == nil {
				t.Fields = map[string]any{}
			}
			t.Fields["_routed_by"] = i
			return
		}
	}
}

// matchRule returns true if every key in the condition equals the ticket field.
func matchRule(fields map[string]any, cond map[string]any) bool {
	if len(cond) == 0 {
		return false
	}
	for k, want := range cond {
		got, ok := fields[k]
		if !ok || !equal(got, want) {
			return false
		}
	}
	return true
}

func equal(a, b any) bool {
	// Loose equality via stringification to handle JSON number/string drift.
	return toString(a) == toString(b)
}
