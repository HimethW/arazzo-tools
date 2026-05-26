# Arazzo v1.1.0 Support Plan

## Summary

Support Arazzo `1.1.0` across the full extension product: shared TypeScript models, Go LSP parser/validator/completions/navigation, visualizer, CLI/MCP runner, and tests, while preserving `1.0.0` and `1.0.1` compatibility.

Sources checked:
- Arazzo v1.1.0 latest: https://spec.openapis.org/arazzo/latest.html
- Arazzo v1.0.1: https://spec.openapis.org/arazzo/v1.0.1.html

Current repo is missing several v1.1.0 fields and behaviors:
- `$self`
- `asyncapi` source descriptions
- `channelPath`
- `timeout`
- `correlationId`
- `action: send | receive`
- step-level `dependsOn`
- `querystring`
- Selector Object
- expanded Expression Type Object
- `$message`
- richer `$sourceDescriptions` / `$components` runtime expressions
- JSONPath/XPath/JSON Pointer selector and replacement behavior
- AsyncAPI runtime adapter architecture
- message serialization layer for JSON now and Avro/Protobuf/etc. later

## Implementation Phases

### Phase 1: Version, Schema, And Compatibility Foundation

Goal: make the repo understand the Arazzo `1.1.0` document shape without changing execution behavior yet.

Changes:
- Update all TypeScript and Go Arazzo models to accept `arazzo: "1.1.0"` while keeping `1.0.0` and `1.0.1`.
- Add root `$self?: string`.
- Extend `sourceDescriptions.type` from `openapi | arazzo` to `openapi | asyncapi | arazzo`.
- Add new step fields:
  - `channelPath?: string`
  - `timeout?: integer` (non-negative integer, milliseconds — not a float/number)
  - `correlationId?: string`
  - `action?: "send" | "receive"`
  - `dependsOn?: string[]`
- Add `querystring` as a valid parameter location.
- Widen output, parameter value, request body payload, and replacement value types so they can later accept Selector Objects.
- Add new `parameters` field to `SuccessActionObject` (new in v1.1.0): a list of `Parameter Object | Reusable Object` to be passed to a workflow referenced by `workflowId`. The `in` field MUST NOT be used on parameters in this context. This field does not exist on `SuccessActionObject` in v1.0.1 and is absent from the current TypeScript interfaces and LSP structs.
- Add new `parameters` field to `FailureActionObject` (new in v1.1.0, spec §5.8.8.1): identical semantics to `SuccessActionObject.parameters` — a list of `Parameter Object | Reusable Object` passed to the workflow referenced by `workflowId`. The `in` field MUST NOT be used. This field is absent from all current models (TypeScript `FailureActionObject`, LSP `FailureAction`, and CLI `Action` structs).
- Rename `CriterionExpressionObject` / `Criterion Expression Type Object` (v1.0.1 name) to `ExpressionTypeObject` throughout all TypeScript interfaces, LSP parser structs, completion code, and validation code. The v1.1.0 spec renames this object and also adds `jsonpointer` as a new allowed `type` value (v1.0.1 only had `jsonpath` and `xpath`). Additionally: the **existing TypeScript `CriterionExpressionObject` interface has a bug — it names the field `expression` instead of the spec-correct `version`** (this is wrong in both v1.0.1 and v1.1.0 — the spec has always called this field `version`). The rename and field correction must happen together. Audit every reference to the old object name AND the old field name `expression` across `arazzo-designer-core`, `arazzo-designer-lsp`, and the CLI Go models.
- Centralize or closely align the schema shape across:
  - `arazzo-designer-core` TypeScript interfaces
  - LSP parser structs
  - CLI runner Go models
  - visualizer data assumptions
- Note pre-existing LSP gap to fix in this phase: the LSP parser `RequestBody` Go struct (`arazzo-designer-lsp/parser/ast.go`) is missing the `Replacements []Replacement` field that already exists in the CLI model and the TypeScript interfaces. Add it now so the LSP is consistent before Phase 6 extends the replacement object.

Validation in this phase:
- Accept `arazzo: 1.1.0`.
- Accept `$self`.
- Accept `asyncapi`.
- Accept the new fields syntactically.
- Enforce that a step has exactly one target selector:
  - `operationId`
  - `operationPath`
  - `channelPath`
  - `workflowId`

Tests:
- Existing `1.0.0` and `1.0.1` files still parse.
- A minimal `1.1.0` file parses.
- A `1.1.0` file with all new fields parses.
- Invalid versions still fail.
- A step with both `operationId` and `channelPath` fails validation.

Checkpoint:
- The extension can open a v1.1.0 file without breaking.
- No AsyncAPI execution yet.

### Phase 2: LSP Authoring Support

Goal: make v1.1.0 pleasant to write in VS Code.

Changes:
- Update LSP completions:
  - default new snippets to `arazzo: "1.1.0"`
  - `$self`
  - `asyncapi`
  - `channelPath`
  - `action`
  - `send`
  - `receive`
  - `timeout`
  - `correlationId`
  - step-level `dependsOn`
  - `querystring`
  - `targetSelectorType`
  - `$message.payload`
  - `$message.header`
  - `$self`
  - `parameters` on `SuccessActionObject`
  - `parameters` on `FailureActionObject`
- Update LSP validation:
  - validate `$self` as a URI-reference without a fragment (spec §5.8.1.1: `$self` MUST NOT contain a fragment identifier).
  - validate `sourceDescriptions.type`.
  - validate `action` is only `send` or `receive`.
  - validate `timeout` is a non-negative integer in milliseconds.
  - validate `dependsOn` step references — three valid forms are accepted: (1) bare `stepId` (local workflow step), (2) `$workflows.<workflowId>.steps.<stepId>` (step in another workflow in this document), (3) `$sourceDescriptions.<name>.<workflowId>.steps.<stepId>` (step in an external Arazzo document). Any other form is invalid.
  - validate `channelPath` is used for AsyncAPI references.
  - validate `correlationId` is meaningful for AsyncAPI receive-style steps.
  - validate component key naming rules from the spec (`^[a-zA-Z0-9\.\-_]+$`).
  - validate that `successCriteria`, when present, contains at least one Criterion Object (empty array is invalid — new MUST in v1.1.0 §5.8.5.1).
  - validate `SuccessActionObject.parameters`: the `in` field MUST NOT be set on any parameter in this list.
  - validate `FailureActionObject.parameters`: the `in` field MUST NOT be set on any parameter in this list (same rule as SuccessActionObject — spec §5.8.8.1).

Tests:
- Completion tests for every new field and enum value.
- Validation tests for bad `$self`, bad `action`, bad `timeout`, invalid source type, and duplicate target selectors.
- `dependsOn` with a bare local stepId that exists passes; a non-existent stepId fails.
- `dependsOn` with `$workflows.<wf>.steps.<s>` cross-workflow form is accepted and validated.
- `dependsOn` with an unrecognized form (e.g. plain string that is not a valid stepId or expression) fails.
- Empty `successCriteria: []` fails validation.
- `SuccessActionObject.parameters` entry with `in: query` fails validation.
- `FailureActionObject.parameters` entry with `in: header` fails validation.

Checkpoint:
- Users can author v1.1.0 files with useful completions and correct diagnostics.
- Runtime still behaves like before for OpenAPI.

### Phase 3: `$self` And Source Resolution

Goal: correctly resolve source documents in v1.1.0.

Changes:
- Enforce full-document parsing before resolving any references (spec §5.5.1: implementations MUST parse entire documents before resolving references; fragmentary parsing produces undefined behavior). The entire Arazzo document must be loaded and parsed so that `$self` and all source description `url` fields are known before any reference resolution begins. This applies to the LSP loader, the CLI loader, and the RPC client.
- Establish the base URI using RFC3986 §5.1.1–5.1.4 priority order:
  - If `$self` is present and is an **absolute** URI: use it directly as the base URI.
  - If `$self` is present and is a **relative** URI-reference: first resolve it against the next applicable base URI source (retrieval URI, encapsulating entity, or application default per RFC3986 §5.1.2–5.1.4), then use the resulting absolute URI as the base URI.
  - If `$self` is absent: use the retrieval URI (file path or HTTP URL) as the base URI.
- Resolve relative `sourceDescriptions.url` values against the base URI established above.
- When referencing external Arazzo documents, use identity-based matching: if the target document has a `$self` field, the reference MUST match the `$self` URI, not just the retrieval location (spec §5.5.2).
- Preserve current local relative file behavior for v1.0.x files (no `$self` → use file path as base URI).

Tests:
- Relative OpenAPI source resolved from local file path when `$self` is absent.
- Relative source resolved correctly when `$self` is an absolute URI.
- Relative `$self` is first resolved against the retrieval URI, and the resulting absolute URI is then used as the base URI for further relative references.
- Remote-style `$self` plus relative source URL produces the expected absolute URI.
- Two documents referencing the same `$self` URI are treated as the same document (identity over location).
- A document is fully parsed before any reference within it is resolved.
- Existing examples still load.

Checkpoint:
- v1.1.0 source loading is deterministic.
- This prepares for AsyncAPI loading but does not execute AsyncAPI yet.

### Phase 4: Selector Objects And Expression Types

Goal: support the new extraction model used by v1.1.0.

Changes:
- Add `SelectorObject`:
  - `context`
  - `selector`
  - `type`
- Add `ExpressionTypeObject` (renamed from `Criterion Expression Type Object` in v1.0.1; also adds `jsonpointer` as a new `type` value):
  - `type: "jsonpath" | "xpath" | "jsonpointer"` (REQUIRED)
  - `version: string` (REQUIRED when this object is used — validation must reject an ExpressionTypeObject that omits `version`)
  - Allowed `version` values per `type`:
    - `jsonpath`: `rfc9535`, `draft-goessner-dispatch-jsonpath-00`
    - `xpath`: `xpath-31`, `xpath-30`, `xpath-20`, `xpath-10`
    - `jsonpointer`: `rfc6901`
  - When the ExpressionTypeObject is **not present at all**, tooling applies these defaults automatically (these are not defaults within the object — the object always requires both fields):
    - JSONPath default: `rfc9535`
    - XPath default: `xpath-31`
    - JSON Pointer default: `rfc6901`
- Allow Selector Objects anywhere v1.1.0 permits them:
  - workflow outputs
  - step outputs
  - parameter values
  - request body payload values
  - payload replacement values
- Add a shared selector evaluation service used by runner components.
- Keep existing string runtime-expression behavior working.

Selector behavior:
- `jsonpointer` uses RFC6901 pointer semantics.
- `jsonpath` uses the current RFC9535-compatible JSONPath library already present in the CLI.
- `xpath` requires XML/XPath support and should be implemented behind the same selector service.
- Unsupported selector versions should produce clear validation/runtime errors.

Tests:
- Selector Object extracts from `$response.body`.
- Selector Object extracts from `$message.payload` using test message context.
- JSON Pointer selector works.
- JSONPath selector works.
- XPath selector works on XML payload.
- Unsupported selector type/version fails clearly.

Checkpoint:
- The runner can evaluate structured selectors independently of AsyncAPI broker execution.

### Phase 5: Runtime Expression Upgrade

Goal: update the evaluator to support v1.1.0 expressions and more complete criteria logic.

Changes:
- Add support for:
  - `$message.header.*`
  - `$message.payload`
  - `$message.payload#/...`
  - `$self`
  - `$sourceDescriptions.<name>.<id>` — implement the two-step resolution priority defined in spec §5.9.2: (1) match `<id>` against operationId or workflowId in the named source description; (2) only if no match, treat `<id>` as a field name on the Source Description Object itself (e.g., `url`). This priority must be implemented explicitly in the evaluator; ambiguous resolution is not permitted.
  - `$components.successActions.*`
  - `$components.failureActions.*`
- Add embedded expression serialization rules:
  - primitives embed as strings.
  - objects/arrays should serialize consistently, normally as JSON.
  - unresolved expressions should produce useful warnings/errors based on context.
- Replace the simple condition parser with a real expression evaluator supporting:
  - `!`
  - `&&`
  - `||`
  - grouping with parentheses
  - property dereference
  - array indexing
  - numeric comparison
  - string comparison
  - case-insensitive string comparison where the spec requires it

Tests:
- `$message.payload.status == "confirmed"`.
- `$message.header.correlationId`.
- `$self` resolves.
- `$sourceDescriptions.petstore.url` resolves.
- object/array embedded expression serialization.
- compound criteria with `&&`, `||`, `!`, parentheses, and indexing.

Checkpoint:
- Runtime expressions are v1.1.0-ready for both OpenAPI and future AsyncAPI steps.

### Phase 6: Payload Replacement Upgrade

Goal: support v1.1.0 replacement targets and replacement values.

Changes:
- Extend replacement object support with:
  - `targetSelectorType`
  - JSON Pointer targets
  - JSONPath targets
  - XPath targets
  - Selector Object values
- Preserve old JSON Pointer replacement behavior.
- Use the shared selector service for target lookup and replacement values.

Tests:
- Existing JSON Pointer replacement still works.
- JSONPath target replacement works.
- XPath target replacement works for XML payload.
- Replacement value can be:
  - literal
  - runtime expression
  - Selector Object

Checkpoint:
- Request body transformation supports v1.1.0 selector semantics.

### Phase 7: OpenAPI Runtime Preservation And Step Dependencies

Goal: upgrade execution ordering without breaking current REST/OpenAPI workflows.

Changes:
- Preserve current sequential execution when no dependencies are declared.
- Honor explicit step `dependsOn`.
- Infer dependencies from expressions like `$steps.<stepId>.outputs.<name>` where safe.
- Detect impossible dependency graphs:
  - missing step ID (local, cross-workflow via `$workflows.*`, or cross-document via `$sourceDescriptions.*`)
  - circular dependencies
  - dependency that never completes
- Note: `dependsOn` establishes a prerequisite relationship only — it does NOT trigger execution of the referenced steps. The runner must not re-execute an already-completed prerequisite step when a later step's `dependsOn` lists it. The runner waits for the depended-on step to complete if it has not yet done so.
- Keep `onSuccess`, `onFailure`, `goto`, `end`, and `retry` behavior compatible with current behavior.
- Clarify precedence between retry exhaustion and following failure actions.

Tests:
- Existing OpenAPI examples still run.
- Explicit `dependsOn` waits for required steps.
- `dependsOn` with a cross-workflow reference (`$workflows.<wf>.steps.<s>`) resolves and waits correctly.
- An already-completed step is not re-executed when another step lists it in `dependsOn`.
- Implicit `$steps.x.outputs.y` dependency is respected.
- Circular dependency fails clearly.
- Retry behavior still works.

Checkpoint:
- REST workflows are still stable.
- Execution engine is ready for async send/receive dependencies.

### Phase 8: AsyncAPI Model Resolution And Visualization

Goal: understand AsyncAPI sources and show them correctly before real broker execution.

Changes:
- Load AsyncAPI source documents from `sourceDescriptions`.
- Index AsyncAPI operations and channels.
- Resolve AsyncAPI references from:
  - `operationId`
  - scoped operation IDs such as `$sourceDescriptions.orderEvents.placeOrder`
  - `channelPath`
- Update navigation:
  - OpenAPI operation navigation still works.
  - AsyncAPI operation navigation works.
  - AsyncAPI channel navigation works for `channelPath`.
- Update visualizer:
  - show `$self` in overview.
  - show source type badges: OpenAPI, AsyncAPI, Arazzo.
  - render `send` and `receive` steps distinctly.
  - show `channelPath`, `action`, `correlationId`, `timeout`, and `dependsOn` in the properties panel.
  - draw dependency edges for step `dependsOn`.

Tests:
- AsyncAPI source loads.
- AsyncAPI operation/channel is indexed.
- `channelPath` navigation resolves to the AsyncAPI source.
- Visualizer shows async metadata.
- Dependency edges render without breaking existing success/failure/goto edges.

Checkpoint:
- AsyncAPI files are authorable, navigable, and visualized.
- Real message transport is still not required.

### Phase 9: AsyncAPI Adapter Interface

Goal: add the runtime boundary that lets the runner send and receive messages without hard-coding Kafka/MQTT/RabbitMQ/etc.

Important concept:
- The runner does not implement brokers.
- Kafka, RabbitMQ, MQTT, NATS, WebSocket servers, and cloud queues are external systems.
- The runner only needs adapters/connectors that know how to talk to those systems.

Core interface:
- `Send(channel, message, options)`.
- `Receive(channel, correlationId, timeout, options)`.
- Return a normalized message object with:
  - headers
  - payload
  - raw body/bytes when needed
  - content type
  - metadata such as topic/queue/channel name

Runner behavior:
- For `action: send`:
  - resolve AsyncAPI operation/channel.
  - evaluate parameters and request body.
  - serialize message.
  - call adapter `Send`.
  - store send metadata.
- For `action: receive`:
  - resolve AsyncAPI operation/channel.
  - evaluate `correlationId`.
  - call adapter `Receive`.
  - enforce `timeout`.
  - expose received message as `$message`.
  - evaluate `successCriteria`.
  - extract outputs from `$message`.

Initial adapters:
- Add an in-memory/test adapter.
- Add clear runtime error when a real broker adapter is required but not configured:
  `AsyncAPI execution requires a configured adapter for this protocol`.

Tests:
- In-memory send succeeds.
- In-memory receive gets a matching message.
- Receive ignores non-matching correlation IDs.
- Receive times out.
- `$message.payload` criteria and outputs work.

Checkpoint:
- AsyncAPI execution can be tested end-to-end without Kafka/MQTT/RabbitMQ.
- Production broker support is ready to be added as separate adapters.

### Phase 10: Message Serialization Layer

Goal: separate message shape from wire format so broker adapters do not each invent serialization logic.

Architecture:
- Arazzo step builds a logical message:
  - headers
  - payload
  - correlation ID
  - content type
- Serializer turns the logical payload into bytes/string.
- Broker adapter sends or receives the bytes/string.
- Deserializer turns received bytes/string back into `$message.payload`.

Initial serializer support:
- JSON first.
- Plain text second if trivial.
- Binary passthrough only when explicitly configured.

Future serializer support:
- Avro.
- Protobuf.
- CloudEvents.
- Custom content types through a serializer registry.

Serializer registry:
- Map content type or AsyncAPI message binding metadata to a serializer.
- Example:
  - `application/json` -> JSON serializer.
  - `text/plain` -> text serializer.
  - `application/x-protobuf` -> Protobuf serializer.
  - `application/avro` -> Avro serializer.

Protobuf notes:
- Protobuf requires schema/message type information.
- The adapter/serializer must know where the `.proto` descriptor or generated type comes from.
- If the AsyncAPI document does not provide enough schema information, runtime should fail clearly instead of guessing.

Avro notes:
- Avro requires an Avro schema or schema registry.
- Schema registry configuration should be adapter/serializer configuration, not core runner logic.

Tests:
- JSON send serializes payload correctly.
- JSON receive deserializes into `$message.payload`.
- Unsupported content type fails clearly.
- Serializer registry selects the correct serializer.
- Placeholder tests document expected behavior for Protobuf/Avro until implemented.

Checkpoint:
- AsyncAPI runtime has the right architecture for JSON now and Protobuf/Avro later.

### Phase 11: First Real Broker Adapter

Goal: add one production-grade broker adapter after the generic runtime is proven.

Recommended first adapter:
- Choose based on target users.
- Good candidates:
  - WebSocket: easiest for demos and local testing.
  - Kafka: common enterprise event-streaming case.
  - MQTT: useful for IoT/event scenarios.

Kafka adapter responsibilities:
- Map AsyncAPI channel to Kafka topic.
- Publish messages to topic.
- Consume messages from topic.
- Support consumer group configuration.
- Match `correlationId` from headers or payload.
- Use serializer registry.
- Support auth/TLS configuration.

MQTT adapter responsibilities:
- Map AsyncAPI channel to MQTT topic.
- Publish to topic.
- Subscribe to topic.
- Match `correlationId`.
- Support QoS where configured.
- Use serializer registry.
- Support auth/TLS configuration.

RabbitMQ adapter responsibilities:
- Map AsyncAPI channel to exchange/routing key/queue.
- Publish AMQP message.
- Consume from queue.
- Match `correlationId`.
- Use serializer registry.
- Support auth/TLS configuration.

Tests:
- Adapter-specific integration tests behind opt-in environment variables.
- Unit tests with mocked broker clients.
- End-to-end sample workflow for the chosen broker.

Checkpoint:
- One real AsyncAPI protocol works in production-like conditions.

### Phase 12: CLI, MCP, Documentation, And Samples

Goal: make the feature usable and explainable.

Changes:
- Update CLI workflow details to include:
  - Arazzo version
  - `$self`
  - source types
  - async channel/action metadata
  - adapter configuration status
- Update MCP responses:
  - include async metadata.
  - surface unsupported adapter errors clearly.
  - show timeout/correlation errors clearly.
- Add examples:
  - minimal v1.1.0 OpenAPI-only workflow.
  - v1.1.0 AsyncAPI send/receive workflow using in-memory adapter.
  - selector object examples.
  - JSONPath/XPath replacement examples.
  - future broker adapter example once one real adapter exists.
- Update user docs:
  - REST vs AsyncAPI explanation.
  - `send` / `receive` explanation.
  - channels explanation.
  - broker vs adapter explanation.
  - serializer explanation.

Tests:
- CLI still lists/runs old workflows.
- CLI reports async adapter errors clearly.
- MCP tool output remains stable for old workflows.
- New examples parse and validate.

Checkpoint:
- Users can understand and try v1.1.0 features safely.

## Final Acceptance Criteria

- `arazzo: 1.1.0` is accepted everywhere.
- Old `1.0.0` and `1.0.1` files still work.
- `$self` and v1.1.0 source resolution work.
- `asyncapi` sources load.
- `channelPath`, `action`, `correlationId`, `timeout`, `dependsOn`, and `querystring` are modeled, validated, completed, and visualized.
- Selector Objects and Expression Type Objects work in supported locations.
- JSONPath, XPath, and JSON Pointer selectors/replacements are supported.
- `$message` expressions work for AsyncAPI receive steps.
- OpenAPI execution remains stable.
- AsyncAPI execution works through the in-memory/test adapter.
- Missing real broker adapter errors are clear.
- Serialization is separated from adapters and supports JSON first.
- Protobuf/Avro support is planned through serializer registry, not hard-coded into broker adapters.

## Assumptions

- Scope is full product support: editor, validation, visualization, CLI runner, MCP execution, and tests.
- AsyncAPI execution will be pluggable, not hard-coded to a specific broker in the initial upgrade.
- Real broker adapters will be added incrementally after the adapter interface and serialization layer are stable.
- JSON is the first supported message serialization format.
- Protobuf, Avro, CloudEvents, and custom content types are follow-up serializer implementations.
- Existing Arazzo `1.0.0` and `1.0.1` behavior must remain backward compatible.
