package server

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/arazzo/lsp/parser"
)

// getModel must TELL a client what each step targets. The case that matters is a bare operationId in
// a document declaring more than one source: the Arazzo text names no source, so a client reading
// only that text cannot know which one owns the operation - it can only guess, and guesses wrong.
func TestGetModelAnnotatesStepTargets(t *testing.T) {
	dir := t.TempDir()
	writeSpecs(t, dir)

	arazzo := `arazzo: "1.1.0"
info:
  title: Targets
  version: "1.0.0"
sourceDescriptions:
  - name: catalog
    url: ./catalog.openapi.yaml
    type: openapi
  - name: orderEvents
    url: ./order-events.asyncapi.yaml
    type: asyncapi
workflows:
  - workflowId: mixed
    steps:
      - stepId: bareRestOperation
        operationId: getProducts
      - stepId: bareAsyncOperation
        operationId: placeOrder
      - stepId: scopedAsyncOperation
        operationId: $sourceDescriptions.orderEvents.onOrderConfirmed
      - stepId: viaChannelPath
        channelPath: orderEvents#/channels/orders
        action: send
      - stepId: viaOperationPath
        operationPath: orderEvents#/operations/placeOrder
      - stepId: nested
        workflowId: other
      - stepId: unresolvable
        operationId: noSuchOperation
`
	s := NewServer()
	uri := openDoc(t, s, filepath.Join(dir, "flow.arazzo.yaml"), arazzo)

	raw, err := s.GetModel(context.Background(), &GetModelParams{URI: string(uri)})
	if err != nil {
		t.Fatalf("GetModel: %v", err)
	}
	doc, ok := raw.(*parser.ArazzoDocument)
	if !ok {
		t.Fatalf("GetModel returned %T, want *parser.ArazzoDocument", raw)
	}
	if len(doc.Workflows) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(doc.Workflows))
	}

	got := map[string]string{}
	for _, step := range doc.Workflows[0].Steps {
		got[step.StepID] = step.StepType
	}

	want := map[string]string{
		// A bare operationId is the whole point: only a lookup inside the declared specs can say
		// which source owns it. Both of these are bare, and they resolve to DIFFERENT types.
		"bareRestOperation":  SourceTypeOpenAPI,
		"bareAsyncOperation": SourceTypeAsyncAPI,
		// The forms that name their source resolve too, through the same index.
		"scopedAsyncOperation": SourceTypeAsyncAPI,
		"viaOperationPath":     SourceTypeAsyncAPI,
		// channelPath is an AsyncAPI channel by definition.
		"viaChannelPath": SourceTypeAsyncAPI,
		"nested":         "workflow",
		// Nothing owns it, so the server says nothing rather than guessing.
		"unresolvable": "",
	}
	for stepID, wantType := range want {
		if got[stepID] != wantType {
			t.Errorf("step %q: stepType = %q, want %q", stepID, got[stepID], wantType)
		}
	}
}

// The annotation must never invent a type for a document whose sources cannot be read: an
// unresolvable target stays empty so a client can fall back rather than display a wrong label.
func TestGetModelLeavesUnresolvableStepsUnannotated(t *testing.T) {
	dir := t.TempDir()

	arazzo := `arazzo: "1.1.0"
info:
  title: Remote only
  version: "1.0.0"
sourceDescriptions:
  - name: remote
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: wf
    steps:
      - stepId: unknowable
        operationId: whatever
`
	s := NewServer()
	uri := openDoc(t, s, filepath.Join(dir, "flow.arazzo.yaml"), arazzo)

	raw, err := s.GetModel(context.Background(), &GetModelParams{URI: string(uri)})
	if err != nil {
		t.Fatalf("GetModel: %v", err)
	}
	doc := raw.(*parser.ArazzoDocument)
	if got := doc.Workflows[0].Steps[0].StepType; got != "" {
		t.Errorf("stepType = %q for an unreachable remote source, want empty", got)
	}
}
