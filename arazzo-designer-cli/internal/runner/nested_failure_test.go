package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wso2/arazzo-designer-cli/internal/failure"
	"github.com/wso2/arazzo-designer-cli/internal/models"
	"github.com/wso2/arazzo-designer-cli/internal/telemetry"
)

// kafkaBus is a source with no adapter, so any step against it fails with a known class and no
// network access.
const kafkaBus = `asyncapi: 3.0.0
info:
  title: Bus
  version: 1.0.0
servers:
  cluster:
    host: kafka.example.com:9092
    protocol: kafka
channels:
  events:
    address: events
`

// localBus runs in-process, so a step against it succeeds with no network access.
const localBus = `asyncapi: 3.0.0
info:
  title: Local
  version: 1.0.0
channels:
  events:
    address: local/events
`

func writeFlow(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range map[string]string{
		"kafka.asyncapi.yaml": kafkaBus,
		"local.asyncapi.yaml": localBus,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p := filepath.Join(dir, "flow.arazzo.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// A nested workflow that fails must fail the step that called it, and the parent with it. A nested
// step records no status of its own, so without explicit propagation the parent reported
// workflow_complete with no error at all - a failure one level down vanished from /run and MCP.
func TestNestedWorkflowFailureFailsTheParent(t *testing.T) {
	p := writeFlow(t, `arazzo: 1.1.0
info:
  title: T
  version: "1.0.0"
sourceDescriptions:
  - name: kafkaBus
    url: ./kafka.asyncapi.yaml
    type: asyncapi
workflows:
  - workflowId: parent
    steps:
      - stepId: callChild
        workflowId: child
  - workflowId: child
    steps:
      - stepId: boom
        channelPath: kafkaBus#/channels/events
        action: send
`)
	r, err := NewArazzoRunner(p, &models.RuntimeParams{}, &telemetry.NoopSink{})
	if err != nil {
		t.Fatal(err)
	}

	got := r.ExecuteWorkflow("parent", nil)
	if got.Status != models.WorkflowStatusError {
		t.Errorf("parent status = %v, want error - a failed nested workflow must not report complete", got.Status)
	}
	// The nested failure's OWN class, not a wrapper saying only "a nested workflow failed".
	if got.ErrorClass != string(failure.AdapterUnsupported) {
		t.Errorf("parent ErrorClass = %q, want %q", got.ErrorClass, failure.AdapterUnsupported)
	}
	if !strings.Contains(got.Error, "callChild") {
		t.Errorf("parent Error should name the calling step, got: %q", got.Error)
	}
}
