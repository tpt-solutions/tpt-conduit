// Package examples ships the reference workflow templates described in the
// spec (IT helpdesk, HR onboarding, asset tracking, generic approval chain).
// They double as end-to-end fixtures for engine tests.
package examples

import (
	_ "embed"

	"tptconduit/engine"
	"tptconduit/engine/dsl"
)

//go:embed helpdesk.yaml
var helpdeskYAML []byte

//go:embed approval_chain.yaml
var approvalChainYAML []byte

//go:embed hr_onboarding.yaml
var hrOnboardingYAML []byte

//go:embed asset_tracking.yaml
var assetTrackingYAML []byte

// Helpdesk returns the IT helpdesk ticket workflow definition.
func Helpdesk() engine.WorkflowDef { return must(helpdeskYAML) }

// ApprovalChain returns a generic multi-step approval workflow.
func ApprovalChain() engine.WorkflowDef { return must(approvalChainYAML) }

// HROnboarding returns the HR onboarding workflow.
func HROnboarding() engine.WorkflowDef { return must(hrOnboardingYAML) }

// AssetTracking returns the asset tracking workflow.
func AssetTracking() engine.WorkflowDef { return must(assetTrackingYAML) }

func must(b []byte) engine.WorkflowDef {
	w, err := dsl.ParseYAML(b)
	if err != nil {
		panic("examples: " + err.Error())
	}
	return w
}
