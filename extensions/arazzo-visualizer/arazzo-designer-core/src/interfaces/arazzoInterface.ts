/* eslint-disable @typescript-eslint/no-explicit-any */
/**
 * Copyright (c) 2025, WSO2 LLC. (https://www.wso2.com) All Rights Reserved.
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

export interface ArazzoDefinition {
    arazzo: '1.0.0' | '1.0.1' | '1.1.0';

    // Optional URL reference to the Arazzo description itself (v1.1.0)
    $self?: string;

    //Metadata about the API workflows
    info: ArazzoInfo;

    //List of external API specifications
    sourceDescriptions: SourceDescription[];

    //The core workflows defined
    workflows: ArazzoWorkflow[];

    // Reusable components (inputs, steps, criteria, etc.)
    components?: ComponentsObject;

    //any Extensions
    [key: string]: any;
}

export interface ArazzoInfo {
    title: string;
    version: string;    // (version for the document)
    summary?: string;
    description?: string;
}

export interface SourceDescription {
    name: string;
    url: string;
    type?: 'openapi' | 'asyncapi' | 'arazzo'; // optional per spec §5.8.3; asyncapi added in v1.1.0

    // Optional headers required to fetch the source spec(auth)
    'x-headers'?: Record<string, string>;
}


export interface ArazzoWorkflow {
    workflowId: string;
    summary?: string;
    description?: string;

    //The data required to start the workflow.
    //JSON Schema object

    inputs?: JSONSchema;

    // Global parameters applied to all steps in this workflow
    parameters?: (Parameter | ReusableObject)[];

    dependsOn?: string[];       //any other workflowIds that needs to be done before this one

    // The sequence of steps. 
    // Maps to React Flow Nodes and Edges.
    steps: StepObject[];

    // Assertions to validate overall workflow success.
    successActions?: (SuccessActionObject | ReusableObject)[];

    // Assertions that indicate workflow failure
    failureActions?: (FailureActionObject | ReusableObject)[];

    // Data exposed after the workflow finishes.
    // Maps internal step data to external output variables.
    // Example: { "finalizedPaymentPlan": "$steps.retrieveFinalizedPaymentPlan.finalizedPaymentPlan" }
    outputs?: Record<string, string | SelectorObject | any>;

}


export interface StepObject {
    /** Unique ID for this step (used in 'dependsOn' and runtime expressions) */
    stepId: string;

    description?: string;

    /** * Links the step to a specific API endpoint.
     * Example: "findEligibleProducts"
     */
    operationId?: string;

    /** Alternative to operationId using JSONPointer/XPath */
    operationPath?: string;

    /** If this step calls another nested Arazzo workflow */
    workflowId?: string;

    /** v1.1.0: Reference to an AsyncAPI channel (alternative to operationId/operationPath/workflowId) */
    channelPath?: string;

    /** v1.1.0: Maximum time in milliseconds the step is allowed to run */
    timeout?: number;

    /** v1.1.0: AsyncAPI correlation ID expression */
    correlationId?: string;

    /** v1.1.0: AsyncAPI message direction for channelPath steps */
    action?: 'send' | 'receive';

    /** v1.1.0: Step-level prerequisites — stepIds that must complete before this step */
    dependsOn?: string[];

    /** * Parameters passed to the operation (query, path, header, cookie).
     * Example: loanTransactionId in path
     */
    parameters?: (Parameter | ReusableObject)[];

    /** * The body sent with the request.
     * Can contain Runtime Expressions like "{$inputs.customer}"
     */
    requestBody?: RequestBody;

    /** * Immediate assertions to validate this specific step.
     * Example: "$statusCode == 200"
     */
    successCriteria?: Criterion[];

    /** * Branching Logic: What to do when the step succeeds.
     * Used for "If eligible -> goto createCustomer, Else -> end"
     */
    onSuccess?: (SuccessActionObject | ReusableObject)[];


    /** * Branching Logic: What to do when the step fails.
     */
    onFailure?: (FailureActionObject | ReusableObject)[];
    /** * Output mapping.
     * Stores parts of the response into variables for later steps.
     */
    outputs?: Record<string, string | SelectorObject | any>;
}

export interface SuccessActionObject {
    name: string;
    type: 'goto' | 'end'; // Possible action types
    workflowId?: string; // For 'goto' actions
    stepId?: string; // For 'goto' actions
    parameters?: (Parameter | ReusableObject)[]; // v1.1.0: parameters passed to the referenced workflow
    criteria?: Criterion[]; // Optional criteria to evaluate before action
}

export interface FailureActionObject {
    name: string;
    type: 'goto' | 'end' | 'retry'; // Possible action types
    workflowId?: string; // For 'goto' or 'retry' actions
    stepId?: string; // For 'goto' or 'retry' actions
    parameters?: (Parameter | ReusableObject)[]; // v1.1.0: parameters passed to the referenced workflow
    retryAfter?: number; // for 'retry' actions
    retryLimit?: number; // for 'retry' actions
    criteria?: Criterion[]; // Optional criteria to evaluate before action
}

export interface Criterion {
    //The logic to evaluate. (runtime expressions)

    condition: string;

    context?: string;       //must be provided if type is specified as jsonpath

    //Defaults to 'simple' if not specified
    type?: 'regex' | 'jsonpath' | 'simple' | 'xpath' | ExpressionTypeObject;
}

/** v1.1.0: Renamed from CriterionExpressionObject. Specifies the expression dialect and version. */
export interface ExpressionTypeObject {
    type: 'jsonpath' | 'xpath' | 'jsonpointer';
    // jsonpath: 'draft-goessner-dispatch-jsonpath-00' | 'rfc9535'
    // xpath:    'xpath-10' | 'xpath-20' | 'xpath-30' | 'xpath-31'
    // jsonpointer: 'rfc6901'
    version: 'draft-goessner-dispatch-jsonpath-00' | 'rfc9535' | 'xpath-10' | 'xpath-20' | 'xpath-30' | 'xpath-31' | 'rfc6901';
}

export interface Parameter {
    name: string;
    in?: 'header' | 'query' | 'querystring' | 'path' | 'cookie'; // optional per spec §5.8.6; omitted when workflowId context; 'querystring' added v1.1.0
    //description?: string;
    //required?: boolean;
    value: string | number | boolean | SelectorObject | any;  //can be a runtime expression, raw value, or Selector Object
    //schema?: JSONSchema;
}

export interface RequestBody {
    contentType?: string; // e.g., "application/json"
    // The payload can be a raw object, a stringified JSON with injected variables, or Selector Objects.
    payload?: any; // optional per spec §5.8.14; replacements alone may be sufficient
    replacements?: PayloadReplacementObject[];
}

export interface PayloadReplacementObject {
    // JSON Pointer, XPath, or JSONPath to locate the field in the payload
    target: string;
    // Selector type for target — optional; defaults to JSON Pointer (JSON) or XPath (XML) per spec §5.8.15
    targetSelectorType?: string | ExpressionTypeObject;
    // The value or runtime expression or Selector Object to inject
    value: string | number | boolean | SelectorObject | any;
}

/** v1.1.0 §5.8.13: Fine-grained data selection from structured data using JSONPath, XPath, or JSON Pointer */
export interface SelectorObject {
    /** REQUIRED. Runtime expression evaluating to structured data (e.g. $response.body) */
    context: string;
    /** REQUIRED. Selector expression (JSONPath, XPath, or JSON Pointer) */
    selector: string;
    /** REQUIRED. Type of selector expression */
    type: 'jsonpath' | 'xpath' | 'jsonpointer' | ExpressionTypeObject;
}

// Reusable Components (For complex specs)

export interface ComponentsObject {
    inputs?: Record<string, JSONSchema>;
    parameters?: Record<string, Parameter>;
    successActions?: Record<string, SuccessActionObject | ReusableObject>;
    failureActions?: Record<string, FailureActionObject | ReusableObject>;
}

export interface ReusableObject {
    reference: string; // e.g., "#/components/parameters/param1"
    value?: string; // Optional overrides
}

// JSON Schema Helper (For complex Inputs)

// Simplified JSON Schema interface for Workflow Inputs.
// This matches the "inputs" section

export interface JSONSchema {
    type?: string; // "object", "string", "array", etc.
    required?: string[];
    properties?: Record<string, JSONSchema>;
    items?: JSONSchema; // For arrays
    oneOf?: JSONSchema[];
    anyOf?: JSONSchema[];
    description?: string;
    format?: string;
    pattern?: string;
    minLength?: number;
    maxLength?: number;
    [key: string]: any;
}