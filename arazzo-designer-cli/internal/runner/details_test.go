package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wso2/arazzo-designer-cli/internal/models"
	"github.com/wso2/arazzo-designer-cli/internal/telemetry"
)

// An AsyncAPI source with a real transport, one channel reached two ways, and everything the
// document can declare about its messages.
const detailsBus = `asyncapi: 3.0.0
info:
  title: Bus
  version: 1.0.0
servers:
  broker:
    host: broker.example.com
    protocol: mqtt
channels:
  orders:
    address: orders/new
    messages:
      order:
        contentType: application/json
        correlationId:
          location: $message.header#/correlationId
operations:
  onOrder:
    action: receive
    channel:
      $ref: '#/channels/orders'
`

const detailsKafka = `asyncapi: 3.0.0
info:
  title: Stream
  version: 1.0.0
servers:
  cluster:
    host: kafka.example.com:9092
    protocol: kafka
channels:
  events:
    address: events
`

func writeDetailsDoc(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// GetWorkflowDetails must describe a step the way it will actually RUN — resolving through the same
// helpers the executor uses — rather than re-deriving anything from the step text.
func TestGetWorkflowDetailsDescribesAsyncSteps(t *testing.T) {
	dir := t.TempDir()
	writeDetailsDoc(t, dir, "bus.asyncapi.yaml", detailsBus)
	writeDetailsDoc(t, dir, "stream.asyncapi.yaml", detailsKafka)

	arazzo := writeDetailsDoc(t, dir, "wf.arazzo.yaml", `arazzo: 1.1.0
$self: ./wf.arazzo.yaml
info:
  title: T
  version: "1.0.0"
sourceDescriptions:
  - name: bus
    url: ./bus.asyncapi.yaml
    type: asyncapi
  - name: stream
    url: ./stream.asyncapi.yaml
    type: asyncapi
workflows:
  - workflowId: wf
    steps:
      - stepId: viaChannel
        channelPath: bus#/channels/orders
        action: send
      - stepId: viaOperationId
        operationId: onOrder
      - stepId: unsupported
        channelPath: stream#/channels/events
        action: send
`)

	r, err := NewArazzoRunner(arazzo, &models.RuntimeParams{}, &telemetry.NoopSink{})
	if err != nil {
		t.Fatalf("runner: %v", err)
	}
	d := r.GetWorkflowDetails("wf")
	if d == nil {
		t.Fatal("no details returned")
	}

	// Document level.
	if d["arazzoVersion"] != "1.1.0" {
		t.Errorf("arazzoVersion: got %v", d["arazzoVersion"])
	}
	if d["self"] != "./wf.arazzo.yaml" {
		t.Errorf("self: got %v", d["self"])
	}
	if srcs, _ := d["sourceDescriptions"].([]map[string]interface{}); len(srcs) != 2 {
		t.Errorf("expected both sources described, got %v", d["sourceDescriptions"])
	}

	steps := map[string]map[string]interface{}{}
	for _, raw := range d["steps"].([]map[string]interface{}) {
		steps[raw["stepId"].(string)] = raw
	}

	// A channelPath step: type, transport and the channel's declarations.
	ch := steps["viaChannel"]
	if ch["stepType"] != "asyncapi" || ch["adapter"] != "mqtt" || ch["channel"] != "orders/new" {
		t.Errorf("viaChannel: got type=%v adapter=%v channel=%v", ch["stepType"], ch["adapter"], ch["channel"])
	}
	if types, _ := ch["contentTypes"].([]string); len(types) != 1 || types[0] != "application/json" {
		t.Errorf("viaChannel contentTypes: got %v", ch["contentTypes"])
	}
	if locs, _ := ch["correlationIdLocations"].([]string); len(locs) != 1 {
		t.Errorf("viaChannel correlationIdLocations: got %v", ch["correlationIdLocations"])
	}

	// The case that cannot be derived from the step text: no channelPath at all, and the direction
	// comes from the operation rather than an `action` the step never wrote.
	op := steps["viaOperationId"]
	if op["stepType"] != "asyncapi" {
		t.Errorf("an operationId resolving to an AsyncAPI operation is async, got %v", op["stepType"])
	}
	if op["channel"] != "orders/new" || op["action"] != "receive" {
		t.Errorf("viaOperationId: got channel=%v action=%v", op["channel"], op["action"])
	}

	// A protocol that cannot run says so, without building an adapter.
	bad := steps["unsupported"]
	if bad["adapter"] != nil {
		t.Errorf("kafka should report no adapter, got %v", bad["adapter"])
	}
	if msg, _ := bad["adapterError"].(string); msg == "" {
		t.Error("kafka should report why it cannot run")
	}
}

// A REST-only document must describe exactly as it did before: no async keys anywhere.
func TestGetWorkflowDetailsLeavesRestStepsAlone(t *testing.T) {
	dir := t.TempDir()
	writeDetailsDoc(t, dir, "api.openapi.yaml", `openapi: 3.0.0
info:
  title: API
  version: 1.0.0
paths:
  /things:
    get:
      operationId: listThings
      responses:
        '200':
          description: ok
`)
	arazzo := writeDetailsDoc(t, dir, "wf.arazzo.yaml", `arazzo: 1.1.0
info:
  title: T
  version: "1.0.0"
sourceDescriptions:
  - name: api
    url: ./api.openapi.yaml
    type: openapi
workflows:
  - workflowId: wf
    steps:
      - stepId: list
        operationId: listThings
`)
	r, err := NewArazzoRunner(arazzo, &models.RuntimeParams{}, &telemetry.NoopSink{})
	if err != nil {
		t.Fatalf("runner: %v", err)
	}
	step := r.GetWorkflowDetails("wf")["steps"].([]map[string]interface{})[0]

	if step["stepType"] != "openapi" {
		t.Errorf("a REST step should be openapi, got %v", step["stepType"])
	}
	for _, k := range []string{"adapter", "adapterError", "channel", "channelPath", "action", "contentTypes", "correlationIdLocations"} {
		if _, present := step[k]; present {
			t.Errorf("REST step gained async key %q: %v", k, step[k])
		}
	}
	// The pre-existing keys must survive untouched.
	for _, k := range []string{"stepId", "operationId", "operationPath", "workflowId", "description"} {
		if _, present := step[k]; !present {
			t.Errorf("existing key %q disappeared", k)
		}
	}
}
