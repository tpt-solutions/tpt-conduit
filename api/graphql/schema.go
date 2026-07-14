// Package graphql exposes the TPT Conduit engine over a GraphQL API. The schema
// mirrors the durable execution surface: queries for tickets, workflows, runs
// and the event timeline, plus mutations for creating tickets and driving
// approvals/cancellation.
package graphql

import (
	"sort"

	gql "github.com/graphql-go/graphql"
	ast "github.com/graphql-go/graphql/language/ast"

	"tptconduit/engine"
)

// JSON is a pass-through scalar that carries arbitrary JSON values (ticket
// fields, step output, event payloads) without forcing a fixed shape.
var JSON = gql.NewScalar(gql.ScalarConfig{
	Name: "JSON",
	Serialize: func(v any) any {
		return v
	},
	ParseValue: func(v any) any {
		return v
	},
	ParseLiteral: func(v ast.Value) any {
		return parseLiteral(v)
	},
})

// parseLiteral converts an AST value into a plain Go value (maps/slices/scalars)
// so JSON arguments and JSON output round-trip cleanly.
func parseLiteral(v ast.Value) any {
	switch n := v.(type) {
	case *ast.IntValue:
		return n.Value
	case *ast.FloatValue:
		return n.Value
	case *ast.StringValue:
		return n.Value
	case *ast.BooleanValue:
		return n.Value
	case *ast.EnumValue:
		return n.Value
	case *ast.ListValue:
		out := make([]any, 0, len(n.Values))
		for _, item := range n.Values {
			out = append(out, parseLiteral(item))
		}
		return out
	case *ast.ObjectValue:
		out := map[string]any{}
		for _, f := range n.Fields {
			out[f.Name.Value] = parseLiteral(f.Value)
		}
		return out
	}
	return nil
}

// sortStepStates orders step states by name for deterministic GraphQL output.
func sortStepStates(s []*engine.StepState) {
	sort.Slice(s, func(i, j int) bool { return s[i].Name < s[j].Name })
}

var approverType = gql.NewObject(gql.ObjectConfig{
	Name: "Approver",
	Fields: gql.Fields{
		"role": &gql.Field{Type: gql.NewNonNull(gql.String)},
		"user": &gql.Field{Type: gql.String},
	},
})

var approvalDefType = gql.NewObject(gql.ObjectConfig{
	Name: "Approval",
	Fields: gql.Fields{
		"chain": &gql.Field{
			Type: gql.NewNonNull(gql.NewList(gql.NewNonNull(approverType))),
			Resolve: func(p gql.ResolveParams) (any, error) {
				switch a := p.Source.(type) {
				case *engine.ApprovalDef:
					return a.Chain, nil
				case engine.ApprovalDef:
					return a.Chain, nil
				}
				return nil, nil
			},
		},
	},
})

var stepType = gql.NewObject(gql.ObjectConfig{
	Name: "Step",
	Fields: gql.Fields{
		"name":     &gql.Field{Type: gql.NewNonNull(gql.String)},
		"kind":     &gql.Field{Type: gql.NewNonNull(gql.String)},
		"task":     &gql.Field{Type: gql.String},
		"next":     &gql.Field{Type: gql.String},
		"onError":  &gql.Field{Type: gql.String},
		"assignTo": &gql.Field{Type: gql.String},
		"approval": &gql.Field{Type: approvalDefType},
		"delay":    &gql.Field{Type: JSON, Description: "delay step duration, in nanoseconds"},
		"retry":    &gql.Field{Type: JSON, Description: "retry policy as raw object"},
	},
})

var workflowType = gql.NewObject(gql.ObjectConfig{
	Name: "Workflow",
	Fields: gql.Fields{
		"name":        &gql.Field{Type: gql.NewNonNull(gql.String)},
		"version":     &gql.Field{Type: gql.NewNonNull(gql.String)},
		"description": &gql.Field{Type: gql.String},
		"initial":     &gql.Field{Type: gql.String},
		"steps": &gql.Field{
			Type: gql.NewNonNull(gql.NewList(gql.NewNonNull(stepType))),
			Resolve: func(p gql.ResolveParams) (any, error) {
				w := p.Source.(engine.WorkflowDef)
				return w.Steps, nil
			},
		},
	},
})

var ticketType = gql.NewObject(gql.ObjectConfig{
	Name: "Ticket",
	Fields: gql.Fields{
		"id":              &gql.Field{Type: gql.NewNonNull(gql.String)},
		"workflow":        &gql.Field{Type: gql.NewNonNull(gql.String)},
		"workflowVersion": &gql.Field{Type: gql.NewNonNull(gql.String), Resolve: func(p gql.ResolveParams) (any, error) {
			return p.Source.(*engine.Ticket).WorkflowVer, nil
		}},
		"title":           &gql.Field{Type: gql.NewNonNull(gql.String)},
		"fields":          &gql.Field{Type: JSON},
		"assignee":        &gql.Field{Type: gql.String},
		"queue":           &gql.Field{Type: gql.String},
		"priority":        &gql.Field{Type: gql.String},
		"createdAt":       &gql.Field{Type: gql.NewNonNull(gql.String)},
		"updatedAt":       &gql.Field{Type: gql.NewNonNull(gql.String)},
	},
})

var approvalStateType = gql.NewObject(gql.ObjectConfig{
	Name: "ApprovalState",
	Fields: gql.Fields{
		"chain": &gql.Field{
			Type: gql.NewNonNull(gql.NewList(gql.NewNonNull(approverType))),
			Resolve: func(p gql.ResolveParams) (any, error) {
				return p.Source.(*engine.ApprovalState).Chain, nil
			},
		},
		"index":     &gql.Field{Type: gql.Int},
		"status":    &gql.Field{Type: gql.String},
		"decidedBy": &gql.Field{Type: gql.String},
	},
})

var stepStateType = gql.NewObject(gql.ObjectConfig{
	Name: "StepState",
	Fields: gql.Fields{
		"name":   &gql.Field{Type: gql.NewNonNull(gql.String)},
		"kind":   &gql.Field{Type: gql.String},
		"status": &gql.Field{Type: gql.String},
		"attempt": &gql.Field{Type: gql.Int},
		"output":  &gql.Field{Type: JSON},
		"error":   &gql.Field{Type: gql.String},
		"dueAt":   &gql.Field{Type: gql.String},
		"approval": &gql.Field{
			Type: approvalStateType,
			Resolve: func(p gql.ResolveParams) (any, error) {
				return p.Source.(*engine.StepState).Approval, nil
			},
		},
	},
})

var runType = gql.NewObject(gql.ObjectConfig{
	Name: "WorkflowRun",
	Fields: gql.Fields{
		"id":              &gql.Field{Type: gql.NewNonNull(gql.String)},
		"ticketId":        &gql.Field{Type: gql.NewNonNull(gql.String)},
		"workflow":        &gql.Field{Type: gql.String},
		"workflowVersion": &gql.Field{Type: gql.String, Resolve: func(p gql.ResolveParams) (any, error) {
			return p.Source.(*engine.WorkflowRun).WorkflowVer, nil
		}},
		"status": &gql.Field{Type: gql.String},
		"currentStep": &gql.Field{Type: gql.String, Resolve: func(p gql.ResolveParams) (any, error) {
			return p.Source.(*engine.WorkflowRun).Current, nil
		}},
		"steps": &gql.Field{
			Type: gql.NewNonNull(gql.NewList(gql.NewNonNull(stepStateType))),
			Resolve: func(p gql.ResolveParams) (any, error) {
				run := p.Source.(*engine.WorkflowRun)
				out := make([]*engine.StepState, 0, len(run.Steps))
				for _, s := range run.Steps {
					out = append(out, s)
				}
				// Stable ordering by step name for deterministic responses.
				sortStepStates(out)
				return out, nil
			},
		},
		"output":   &gql.Field{Type: JSON},
		"failed":   &gql.Field{Type: gql.Boolean},
		"createdAt": &gql.Field{Type: gql.String},
		"updatedAt": &gql.Field{Type: gql.String},
	},
})

var eventType = gql.NewObject(gql.ObjectConfig{
	Name: "Event",
	Fields: gql.Fields{
		"seq":     &gql.Field{Type: gql.Int},
		"type":    &gql.Field{Type: gql.NewNonNull(gql.String)},
		"at":      &gql.Field{Type: gql.String},
		"payload": &gql.Field{Type: JSON},
		"scheduleAt": &gql.Field{Type: gql.String},
	},
})

var createTicketInput = gql.NewInputObject(gql.InputObjectConfig{
	Name: "CreateTicketInput",
	Fields: gql.InputObjectConfigFieldMap{
		"workflow": &gql.InputObjectFieldConfig{Type: gql.NewNonNull(gql.String)},
		"version":  &gql.InputObjectFieldConfig{Type: gql.NewNonNull(gql.String)},
		"title":    &gql.InputObjectFieldConfig{Type: gql.NewNonNull(gql.String)},
		"fields":   &gql.InputObjectFieldConfig{Type: JSON},
	},
})

// NewSchema builds the GraphQL schema bound to the given engine.
func NewSchema(e *engine.Engine) (gql.Schema, error) {
	query := gql.NewObject(gql.ObjectConfig{
		Name: "Query",
		Fields: gql.Fields{
			"tickets": &gql.Field{
				Type: gql.NewNonNull(gql.NewList(gql.NewNonNull(ticketType))),
				Resolve: func(p gql.ResolveParams) (any, error) {
					return e.ListTickets(p.Context)
				},
			},
			"ticket": &gql.Field{
				Type: ticketType,
				Args: gql.FieldConfigArgument{
					"id": &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
				},
				Resolve: func(p gql.ResolveParams) (any, error) {
					id, _ := p.Args["id"].(string)
					return e.GetTicket(p.Context, id)
				},
			},
			"workflows": &gql.Field{
				Type: gql.NewNonNull(gql.NewList(gql.NewNonNull(workflowType))),
				Resolve: func(p gql.ResolveParams) (any, error) {
					return e.ListWorkflows(p.Context)
				},
			},
			"workflow": &gql.Field{
				Type: workflowType,
				Args: gql.FieldConfigArgument{
					"name":    &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
					"version": &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
				},
				Resolve: func(p gql.ResolveParams) (any, error) {
					name, _ := p.Args["name"].(string)
					ver, _ := p.Args["version"].(string)
					return e.GetWorkflow(p.Context, name, ver)
				},
			},
			"runs": &gql.Field{
				Type: gql.NewNonNull(gql.NewList(gql.NewNonNull(runType))),
				Resolve: func(p gql.ResolveParams) (any, error) {
					ids, err := e.ListRuns(p.Context)
					if err != nil {
						return nil, err
					}
					runs := make([]*engine.WorkflowRun, 0, len(ids))
					for _, id := range ids {
						r, err := e.GetRun(p.Context, id)
						if err != nil {
							continue
						}
						runs = append(runs, r)
					}
					return runs, nil
				},
			},
			"run": &gql.Field{
				Type: runType,
				Args: gql.FieldConfigArgument{
					"id": &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
				},
				Resolve: func(p gql.ResolveParams) (any, error) {
					id, _ := p.Args["id"].(string)
					return e.GetRun(p.Context, id)
				},
			},
			"events": &gql.Field{
				Type: gql.NewNonNull(gql.NewList(gql.NewNonNull(eventType))),
				Args: gql.FieldConfigArgument{
					"runId": &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
				},
				Resolve: func(p gql.ResolveParams) (any, error) {
					id, _ := p.Args["runId"].(string)
					return e.Events(p.Context, id)
				},
			},
		},
	})

	mutation := gql.NewObject(gql.ObjectConfig{
		Name: "Mutation",
		Fields: gql.Fields{
			"createTicket": &gql.Field{
				Type: gql.NewNonNull(ticketType),
				Args: gql.FieldConfigArgument{
					"input": &gql.ArgumentConfig{Type: gql.NewNonNull(createTicketInput)},
				},
				Resolve: func(p gql.ResolveParams) (any, error) {
					in, _ := p.Args["input"].(map[string]any)
					workflow, _ := in["workflow"].(string)
					version, _ := in["version"].(string)
					title, _ := in["title"].(string)
					fields, _ := in["fields"].(map[string]any)
					t, _, err := e.CreateTicket(p.Context, workflow, version, title, fields)
					if err != nil {
						return nil, err
					}
					return t, nil
				},
			},
			"approve": &gql.Field{
				Type: gql.NewNonNull(gql.Boolean),
				Args: gql.FieldConfigArgument{
					"runId":   &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
					"step":    &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
					"by":      &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
					"comment": &gql.ArgumentConfig{Type: gql.String},
				},
				Resolve: func(p gql.ResolveParams) (any, error) {
					runID, _ := p.Args["runId"].(string)
					step, _ := p.Args["step"].(string)
					by, _ := p.Args["by"].(string)
					comment, _ := p.Args["comment"].(string)
					if err := e.Approve(p.Context, runID, step, by, comment); err != nil {
						return false, err
					}
					return true, nil
				},
			},
			"reject": &gql.Field{
				Type: gql.NewNonNull(gql.Boolean),
				Args: gql.FieldConfigArgument{
					"runId":  &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
					"step":   &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
					"by":     &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
					"reason": &gql.ArgumentConfig{Type: gql.String},
				},
				Resolve: func(p gql.ResolveParams) (any, error) {
					runID, _ := p.Args["runId"].(string)
					step, _ := p.Args["step"].(string)
					by, _ := p.Args["by"].(string)
					reason, _ := p.Args["reason"].(string)
					if err := e.Reject(p.Context, runID, step, by, reason); err != nil {
						return false, err
					}
					return true, nil
				},
			},
			"cancel": &gql.Field{
				Type: gql.NewNonNull(gql.Boolean),
				Args: gql.FieldConfigArgument{
					"runId":  &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
					"reason": &gql.ArgumentConfig{Type: gql.String},
				},
				Resolve: func(p gql.ResolveParams) (any, error) {
					runID, _ := p.Args["runId"].(string)
					reason, _ := p.Args["reason"].(string)
					if err := e.Cancel(p.Context, runID, reason); err != nil {
						return false, err
					}
					return true, nil
				},
			},
		},
	})

	return gql.NewSchema(gql.SchemaConfig{Query: query, Mutation: mutation})
}
