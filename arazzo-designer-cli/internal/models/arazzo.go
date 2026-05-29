// Package models - arazzo.go defines the Arazzo specification document types.
package models

// ArazzoDoc represents a parsed Arazzo specification document.
type ArazzoDoc struct {
	Arazzo             string              `yaml:"arazzo" json:"arazzo"`
	Self               string              `yaml:"$self,omitempty" json:"$self,omitempty"` // v1.1.0: URL reference to the Arazzo description itself
	Info               ArazzoInfo          `yaml:"info" json:"info"`
	SourceDescriptions []SourceDescription `yaml:"sourceDescriptions" json:"sourceDescriptions"`
	Workflows          []Workflow          `yaml:"workflows" json:"workflows"`
	Components         *ArazzoComponents   `yaml:"components,omitempty" json:"components,omitempty"`
}

// ArazzoInfo describes the Arazzo document metadata.
type ArazzoInfo struct {
	Title       string `yaml:"title" json:"title"`
	Summary     string `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Version     string `yaml:"version" json:"version"`
}

// SourceDescription represents an API source referenced by the Arazzo document.
type SourceDescription struct {
	Name string `yaml:"name" json:"name"`
	URL  string `yaml:"url" json:"url"`
	Type string `yaml:"type,omitempty" json:"type,omitempty"` // "openapi", "asyncapi", or "arazzo"
}

// Workflow represents an Arazzo workflow.
type Workflow struct {
	WorkflowID  string                 `yaml:"workflowId" json:"workflowId"`
	Summary     string                 `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Inputs      map[string]interface{} `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	DependsOn   []string               `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`
	Steps       []Step                 `yaml:"steps" json:"steps"`
	Outputs     map[string]interface{} `yaml:"outputs,omitempty" json:"outputs,omitempty"` // widened from map[string]string to support Selector Objects
	Parameters  []Parameter            `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

// Step represents a single step within a workflow.
type Step struct {
	StepID          string                 `yaml:"stepId" json:"stepId"`
	Description     string                 `yaml:"description,omitempty" json:"description,omitempty"`
	OperationID     string                 `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	OperationPath   string                 `yaml:"operationPath,omitempty" json:"operationPath,omitempty"`
	WorkflowID      string                 `yaml:"workflowId,omitempty" json:"workflowId,omitempty"`
	ChannelPath     string                 `yaml:"channelPath,omitempty" json:"channelPath,omitempty"`     // v1.1.0: AsyncAPI channel reference
	Timeout         int                    `yaml:"timeout,omitempty" json:"timeout,omitempty"`             // v1.1.0: max execution time in ms
	CorrelationID   string                 `yaml:"correlationId,omitempty" json:"correlationId,omitempty"` // v1.1.0: AsyncAPI correlation ID
	Action          string                 `yaml:"action,omitempty" json:"action,omitempty"`               // v1.1.0: "send" or "receive"
	DependsOn       []string               `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`         // v1.1.0: step-level prerequisites
	Parameters      []Parameter            `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	RequestBody     *RequestBody           `yaml:"requestBody,omitempty" json:"requestBody,omitempty"`
	SuccessCriteria []Criterion            `yaml:"successCriteria,omitempty" json:"successCriteria,omitempty"`
	OnSuccess       []Action               `yaml:"onSuccess,omitempty" json:"onSuccess,omitempty"`
	OnFailure       []Action               `yaml:"onFailure,omitempty" json:"onFailure,omitempty"`
	Outputs         map[string]interface{} `yaml:"outputs,omitempty" json:"outputs,omitempty"` // widened from map[string]string
}

// Parameter represents an operation parameter.
type Parameter struct {
	Name  string      `yaml:"name" json:"name"`
	In    string      `yaml:"in,omitempty" json:"in,omitempty"`
	Value interface{} `yaml:"value" json:"value"`
}

// RequestBody represents the request body for a step.
type RequestBody struct {
	ContentType  string        `yaml:"contentType,omitempty" json:"contentType,omitempty"`
	Payload      interface{}   `yaml:"payload,omitempty" json:"payload,omitempty"`
	Replacements []Replacement `yaml:"replacements,omitempty" json:"replacements,omitempty"`
}

// Replacement represents a payload replacement (v1.1.0: Payload Replacement Object).
type Replacement struct {
	Target             string      `yaml:"target" json:"target"`
	TargetSelectorType interface{} `yaml:"targetSelectorType,omitempty" json:"targetSelectorType,omitempty"` // v1.1.0: string or ExpressionTypeObject
	Value              interface{} `yaml:"value" json:"value"`
}

// Criterion represents a success/failure criterion.
type Criterion struct {
	Condition string      `yaml:"condition" json:"condition"`
	Context   string      `yaml:"context,omitempty" json:"context,omitempty"`
	Type      interface{} `yaml:"type,omitempty" json:"type,omitempty"` // string ("simple","regex","jsonpath","xpath") or ExpressionTypeObject map for specific dialect versions (v1.1.0)
}

// Action represents an onSuccess or onFailure action.
type Action struct {
	Name       string      `yaml:"name,omitempty" json:"name,omitempty"`
	Type       string      `yaml:"type" json:"type"` // "goto", "end", "retry"
	StepID     string      `yaml:"stepId,omitempty" json:"stepId,omitempty"`
	WorkflowID string      `yaml:"workflowId,omitempty" json:"workflowId,omitempty"`
	RetryAfter float64     `yaml:"retryAfter,omitempty" json:"retryAfter,omitempty"`
	RetryLimit int         `yaml:"retryLimit,omitempty" json:"retryLimit,omitempty"`
	Parameters []Parameter `yaml:"parameters,omitempty" json:"parameters,omitempty"` // v1.1.0: parameters passed to referenced workflow
	Criteria   []Criterion `yaml:"criteria,omitempty" json:"criteria,omitempty"`
}

// ArazzoComponents holds reusable components.
type ArazzoComponents struct {
	Inputs         map[string]interface{} `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Parameters     map[string]Parameter   `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	SuccessActions map[string]Action      `yaml:"successActions,omitempty" json:"successActions,omitempty"` // v1.1.0: reusable success actions
	FailureActions map[string]Action      `yaml:"failureActions,omitempty" json:"failureActions,omitempty"` // v1.1.0: reusable failure actions
}
