package runner

import (
	"strings"
	"testing"

	"github.com/wso2/arazzo-designer-cli/internal/failure"
	"github.com/wso2/arazzo-designer-cli/internal/models"
	"github.com/wso2/arazzo-designer-cli/internal/telemetry"
)

func stepWithDeps(id string, deps ...string) map[string]interface{} {
	d := make([]interface{}, len(deps))
	for i, x := range deps {
		d[i] = x
	}
	return map[string]interface{}{"stepId": id, "dependsOn": d}
}

// The step-level dependsOn completion gate (spec §5.8.5.1): a prerequisite must have completed
// SUCCESSFULLY; no reordering, no triggering.
func TestCheckStepDependencies(t *testing.T) {
	r := &ArazzoRunner{}
	state := models.NewExecutionState("wf", nil, nil, nil)
	state.StepsStatus["okStep"] = models.StepStatusSuccess
	state.StepsStatus["failedStep"] = models.StepStatusFailure
	state.DependencyOutputs["ranWorkflow"] = map[string]interface{}{"x": 1}
	// ranWorkflow ran as a dependency: step "s" succeeded, step "skipped" failed, and step
	// "neverRan" is absent (e.g. jumped over by a goto). The cross-workflow gate must check the
	// SPECIFIC referenced step, not just that the workflow ran.
	state.DependencyStepStatus["ranWorkflow"] = map[string]models.StepStatus{
		"s":       models.StepStatusSuccess,
		"skipped": models.StepStatusFailure,
	}

	cases := []struct {
		name    string
		step    map[string]interface{}
		wantErr bool
	}{
		{"no deps", map[string]interface{}{"stepId": "B"}, false},
		{"satisfied local dep", stepWithDeps("B", "okStep"), false},
		{"unmet local dep (never ran)", stepWithDeps("B", "laterStep"), true},
		{"failed prerequisite does not satisfy", stepWithDeps("B", "failedStep"), true},
		{"multiple deps, one unmet", stepWithDeps("B", "okStep", "laterStep"), true},
		{"cross-workflow step that succeeded", stepWithDeps("B", "$workflows.ranWorkflow.steps.s"), false},
		{"cross-workflow step that was skipped/failed", stepWithDeps("B", "$workflows.ranWorkflow.steps.skipped"), true},
		{"cross-workflow step that never ran", stepWithDeps("B", "$workflows.ranWorkflow.steps.neverRan"), true},
		{"cross-workflow to workflow that didn't run", stepWithDeps("B", "$workflows.nope.steps.s"), true},
		{"malformed cross-workflow ref (no .steps.)", stepWithDeps("B", "$workflows.ranWorkflow.s"), true},
		{"cross-document unsupported", stepWithDeps("B", "$sourceDescriptions.ext.wf.steps.s"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := r.checkStepDependencies(tc.step, state)
			if tc.wantErr && err == nil {
				t.Errorf("expected an error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

// A circular WORKFLOW-level dependsOn must be reported clearly, not crash via infinite recursion.
// The cycle is detected inside executeDependencies before any step executes, so no HTTP is made.
func TestWorkflowDependsOnCycle(t *testing.T) {
	r := &ArazzoRunner{
		Sink: &telemetry.NoopSink{},
		Workflows: []interface{}{
			map[string]interface{}{
				"workflowId": "A",
				"dependsOn":  []interface{}{"B"},
				"steps":      []interface{}{map[string]interface{}{"stepId": "s1"}},
			},
			map[string]interface{}{
				"workflowId": "B",
				"dependsOn":  []interface{}{"A"},
				"steps":      []interface{}{map[string]interface{}{"stepId": "s1"}},
			},
		},
	}
	res := r.ExecuteWorkflow("A", nil)
	if res.Status != models.WorkflowStatusError {
		t.Fatalf("expected error status, got %v", res.Status)
	}
	if !strings.Contains(res.Error, "circular") {
		t.Errorf("expected a 'circular' dependency error, got %q", res.Error)
	}
}

// A self-referential workflow dependsOn is also caught.
func TestWorkflowDependsOnSelfCycle(t *testing.T) {
	r := &ArazzoRunner{
		Sink: &telemetry.NoopSink{},
		Workflows: []interface{}{
			map[string]interface{}{
				"workflowId": "solo",
				"dependsOn":  []interface{}{"solo"},
				"steps":      []interface{}{map[string]interface{}{"stepId": "s1"}},
			},
		},
	}
	res := r.ExecuteWorkflow("solo", nil)
	if res.Status != models.WorkflowStatusError || !strings.Contains(res.Error, "circular") {
		t.Errorf("expected a circular-dependency error, got status=%v err=%q", res.Status, res.Error)
	}
}

// Every way the dependsOn gate can refuse a step is a DIFFERENT situation, and the class has to say
// which: an unmet prerequisite is somebody else's failure to chase, a malformed reference is this
// document to fix, and a cross-document reference is a runner limitation that no edit will resolve.
func TestDependencyGateFailuresAreClassified(t *testing.T) {
	r := &ArazzoRunner{}
	state := models.NewExecutionState("wf", nil, nil, nil)
	state.StepsStatus["failedStep"] = models.StepStatusFailure
	state.DependencyStepStatus["ranWorkflow"] = map[string]models.StepStatus{
		"skipped": models.StepStatusFailure,
	}

	cases := []struct {
		name string
		step map[string]interface{}
		want failure.Class
	}{
		{"local prerequisite failed", stepWithDeps("B", "failedStep"), failure.DependencyUnmet},
		{"local prerequisite never ran", stepWithDeps("B", "laterStep"), failure.DependencyUnmet},
		{"cross-workflow prerequisite failed", stepWithDeps("B", "$workflows.ranWorkflow.steps.skipped"), failure.DependencyUnmet},
		{"prerequisite workflow never ran", stepWithDeps("B", "$workflows.nope.steps.s"), failure.DependencyUnmet},
		{"malformed reference", stepWithDeps("B", "$workflows.ranWorkflow.s"), failure.DocumentInvalid},
		{"cross-document not implemented", stepWithDeps("B", "$sourceDescriptions.ext.wf.steps.s"), failure.UnsupportedFeature},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := r.checkStepDependencies(tc.step, state)
			if err == nil {
				t.Fatal("expected the gate to refuse this step")
			}
			if got := failure.ClassOf(err); got != tc.want {
				t.Errorf("class = %q, want %q (message: %v)", got, tc.want, err)
			}
		})
	}
}

// A workflow that is not in the document, and one with no steps, are different problems and get
// different classes - and both must reach the workflow result a caller reads.
func TestWorkflowLevelFailuresAreClassified(t *testing.T) {
	r := &ArazzoRunner{
		Workflows: []interface{}{map[string]interface{}{"workflowId": "empty", "steps": []interface{}{}}},
		Sink:      &telemetry.NoopSink{},
	}

	if got := r.ExecuteWorkflow("nosuchworkflow", nil); got.ErrorClass != string(failure.TargetUnresolved) {
		t.Errorf("unknown workflow: ErrorClass = %q, want %q", got.ErrorClass, failure.TargetUnresolved)
	}
	if got := r.ExecuteWorkflow("empty", nil); got.ErrorClass != string(failure.DocumentInvalid) {
		t.Errorf("workflow with no steps: ErrorClass = %q, want %q", got.ErrorClass, failure.DocumentInvalid)
	}
}
