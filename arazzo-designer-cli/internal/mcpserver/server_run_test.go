package mcpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/server"

	"github.com/wso2/arazzo-designer-cli/internal/failure"
	"github.com/wso2/arazzo-designer-cli/internal/models"
	"github.com/wso2/arazzo-designer-cli/internal/runner"
	"github.com/wso2/arazzo-designer-cli/internal/telemetry"
)

// buildTestServer constructs a minimal MCPServer whose Runner holds the supplied
// workflow slice.  It bypasses file I/O so tests run without an Arazzo document.
func buildTestServer(workflows []interface{}) *MCPServer {
	arazzoRunner := &runner.ArazzoRunner{
		Workflows: workflows,
		Sink:      &telemetry.NoopSink{},
	}
	mcpSrv := server.NewMCPServer("arazzo-test", "1.0.0")
	return &MCPServer{
		Runner:      arazzoRunner,
		MCPServer:   mcpSrv,
		Port:        0,
		Sink:        &telemetry.NoopSink{},
		lastResults: make(map[string]RunResponse),
	}
}

// testWorkflow builds a minimal workflow map for use in tests.
// inputsDef may be nil (no inputs schema), or a JSON-Schema-style map such as:
//
//	map[string]interface{}{
//	    "required":   []interface{}{"petName"},
//	    "properties": map[string]interface{}{
//	        "petName": map[string]interface{}{"type": "string"},
//	    },
//	}
func testWorkflow(workflowID string, inputsDef map[string]interface{}, hasSteps bool) map[string]interface{} {
	wf := map[string]interface{}{
		"workflowId": workflowID,
	}
	if inputsDef != nil {
		wf["inputs"] = inputsDef
	}
	if hasSteps {
		wf["steps"] = []interface{}{
			map[string]interface{}{"stepId": "step1"},
		}
	}
	return wf
}

// doPost calls handleRun directly with a JSON body and returns the decoded response.
func doPost(t *testing.T, srv *MCPServer, path string, body interface{}) (int, RunResponse) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("failed to encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleRun(w, req)
	res := w.Result()
	var resp RunResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return res.StatusCode, resp
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandleRun_NonPostMethod(t *testing.T) {
	srv := buildTestServer(nil)
	req := httptest.NewRequest(http.MethodGet, "/run/", nil)
	w := httptest.NewRecorder()
	srv.handleRun(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleRun_MissingWorkflowID(t *testing.T) {
	srv := buildTestServer(nil)
	// POST to /run/ with empty body – workflowId absent from both URL and body
	status, resp := doPost(t, srv, "/run/", nil)
	if status != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", status)
	}
	if resp.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", resp.Status)
	}
	if resp.Error == "" {
		t.Error("expected a non-empty error message")
	}
}

func TestHandleRun_WorkflowIDFromBody(t *testing.T) {
	// Body-level workflowId is used when URL path has none.
	// Here the workflow does not exist, so we get a 400 "not found" rather than
	// the generic "workflowId is required" – proving the body fallback worked.
	srv := buildTestServer(nil)
	status, resp := doPost(t, srv, "/run/", map[string]interface{}{
		"workflowId": "no-such-workflow",
	})
	if status != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", status)
	}
	if resp.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", resp.Status)
	}
	if resp.Error == "" {
		t.Error("expected a non-empty error message")
	}
}

func TestHandleRun_UnknownWorkflow(t *testing.T) {
	srv := buildTestServer(nil)
	status, resp := doPost(t, srv, "/run/unknown-workflow", nil)
	if status != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", status)
	}
	if resp.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", resp.Status)
	}
	if resp.Error == "" {
		t.Error("expected a non-empty error message")
	}
}

func TestHandleRun_MissingRequiredInput(t *testing.T) {
	inputsDef := map[string]interface{}{
		"required": []interface{}{"petName"},
		"properties": map[string]interface{}{
			"petName": map[string]interface{}{"type": "string"},
		},
	}
	wf := testWorkflow("create-pet", inputsDef, false)
	srv := buildTestServer([]interface{}{wf})

	// Send request without the required "petName" input
	status, resp := doPost(t, srv, "/run/create-pet", map[string]interface{}{
		"inputs": map[string]interface{}{},
	})
	if status != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", status)
	}
	if resp.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", resp.Status)
	}
	if resp.Error == "" {
		t.Error("expected a non-empty error describing the missing field")
	}
}

func TestHandleRun_WrongInputType(t *testing.T) {
	inputsDef := map[string]interface{}{
		"required": []interface{}{"count"},
		"properties": map[string]interface{}{
			"count": map[string]interface{}{"type": "integer"},
		},
	}
	wf := testWorkflow("list-pets", inputsDef, false)
	srv := buildTestServer([]interface{}{wf})

	// "count" must be integer but we send a string
	status, resp := doPost(t, srv, "/run/list-pets", map[string]interface{}{
		"inputs": map[string]interface{}{
			"count": "not-a-number",
		},
	})
	if status != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", status)
	}
	if resp.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", resp.Status)
	}
	if resp.Error == "" {
		t.Error("expected a non-empty error describing the type mismatch")
	}
}

func TestHandleRun_ExecutionFailure_NoSteps(t *testing.T) {
	// A workflow with satisfied inputs but no steps triggers the runner's
	// "Workflow has no steps" error path without needing a real HTTP backend.
	inputsDef := map[string]interface{}{
		"required": []interface{}{"petName"},
		"properties": map[string]interface{}{
			"petName": map[string]interface{}{"type": "string"},
		},
	}
	wf := testWorkflow("empty-workflow", inputsDef, false /* no steps */)
	srv := buildTestServer([]interface{}{wf})

	status, resp := doPost(t, srv, "/run/empty-workflow", map[string]interface{}{
		"inputs": map[string]interface{}{
			"petName": "Buddy",
		},
	})
	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}
	if resp.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", resp.Status)
	}
	if resp.Error == "" {
		t.Error("expected a non-empty error from the runner")
	}
}

func TestHandleRun_InvalidJSONBody(t *testing.T) {
	srv := buildTestServer(nil)
	req := httptest.NewRequest(http.MethodPost, "/run/any-wf", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleRun(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Failure classes (Phase 12 step 2)
// ---------------------------------------------------------------------------

// A request-level failure is not a workflow failure: it is already an HTTP 400, and it carries no
// class. This also pins the response's BACKWARDS COMPATIBILITY - the raw JSON must not grow an
// error_class key for anything that did not have one before.
func TestHandleRun_RequestFailuresCarryNoErrorClass(t *testing.T) {
	srv := buildTestServer([]interface{}{
		testWorkflow("wf", map[string]interface{}{
			"required":   []interface{}{"token"},
			"properties": map[string]interface{}{"token": map[string]interface{}{"type": "string"}},
		}, true),
	})

	status, resp := doPost(t, srv, "/run/wf", map[string]interface{}{"inputs": map[string]interface{}{}})
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
	if resp.ErrorClass != "" {
		t.Errorf("ErrorClass = %q, want empty for a request-level failure", resp.ErrorClass)
	}

	// The wire form, not just the struct: an old client must see exactly the keys it saw before.
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	if _, has := keys["error_class"]; has {
		t.Errorf("error_class must be omitted when there is no class; got %s", raw)
	}
	for _, want := range []string{"status", "error"} {
		if _, has := keys[want]; !has {
			t.Errorf("response lost the %q key: %s", want, raw)
		}
	}
}

// An async failure must reach the HTTP response WITH its class, end to end: a real Arazzo document,
// a real run, a real POST. This is the path a script against /run actually uses.
func TestHandleRun_AsyncFailureReportsItsClass(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	write("stream.asyncapi.yaml", `asyncapi: 3.0.0
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
`)
	arazzo := write("wf.arazzo.yaml", `arazzo: 1.1.0
info:
  title: T
  version: "1.0.0"
sourceDescriptions:
  - name: stream
    url: ./stream.asyncapi.yaml
    type: asyncapi
workflows:
  - workflowId: emitFlow
    steps:
      - stepId: emit
        channelPath: stream#/channels/events
        action: send
`)

	srv, err := NewMCPServer(arazzo, 0, &models.RuntimeParams{}, &telemetry.NoopSink{})
	if err != nil {
		t.Fatalf("NewMCPServer: %v", err)
	}

	status, resp := doPost(t, srv, "/run/emitFlow", map[string]interface{}{})
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 - a failed workflow is still a successful request", status)
	}
	if resp.Status != "failed" {
		t.Fatalf("Status = %q, want failed", resp.Status)
	}
	if resp.ErrorClass != string(failure.AdapterUnsupported) {
		t.Errorf("ErrorClass = %q, want %q (message: %s)", resp.ErrorClass, failure.AdapterUnsupported, resp.Error)
	}
	// The message is unchanged by classification - it is still there, in full, for a person.
	if !strings.Contains(resp.Error, "not yet supported") {
		t.Errorf("the human-readable message must survive: %q", resp.Error)
	}

	// GET /lastResult returns the cached response, so it inherits the class rather than recomputing.
	req := httptest.NewRequest(http.MethodGet, "/lastResult/emitFlow", nil)
	w := httptest.NewRecorder()
	srv.handleLastResult(w, req)
	var cached RunResponse
	if err := json.NewDecoder(w.Result().Body).Decode(&cached); err != nil {
		t.Fatal(err)
	}
	if cached.ErrorClass != resp.ErrorClass {
		t.Errorf("lastResult ErrorClass = %q, want %q", cached.ErrorClass, resp.ErrorClass)
	}
}
