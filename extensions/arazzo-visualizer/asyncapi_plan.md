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
  - `timeout?: number`
  - `correlationId?: string`
  - `action?: "send" | "receive"`
  - `dependsOn?: string[]`
- Add `querystring` as a valid parameter location.
- Widen output, parameter value, request body payload, and replacement value types so they can later accept Selector Objects.
- Centralize or closely align the schema shape across:
  - `arazzo-designer-core` TypeScript interfaces
  - LSP parser structs
  - CLI runner Go models
  - visualizer data assumptions

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
- Update LSP validation:
  - validate `$self` as a URI-reference without a fragment.
  - validate `sourceDescriptions.type`.
  - validate `action` is only `send` or `receive`.
  - validate `timeout` is a non-negative integer in milliseconds.
  - validate `dependsOn` step IDs exist.
  - validate `channelPath` is used for AsyncAPI references.
  - validate `correlationId` is meaningful for AsyncAPI receive-style steps.
  - validate component key naming rules from the spec.

Tests:
- Completion tests for every new field and enum value.
- Validation tests for bad `$self`, bad `action`, bad `timeout`, invalid `dependsOn`, invalid source type, and duplicate target selectors.

Checkpoint:
- Users can author v1.1.0 files with useful completions and correct diagnostics.
- Runtime still behaves like before for OpenAPI.

### Phase 3: `$self` And Source Resolution

Goal: correctly resolve source documents in v1.1.0.

Changes:
- Parse complete Arazzo documents before resolving external references.
- Use `$self` as the document identity/base URI when present.
- Fall back to the retrieval URI or local file path when `$self` is absent.
- Resolve relative `sourceDescriptions.url` values against `$self` or retrieval URI.
- Match external Arazzo documents by `$self` when present.
- Preserve current local relative file behavior for v1.0.x files.

Tests:
- Relative OpenAPI source resolved from local file path when `$self` is absent.
- Relative source resolved from `$self` when `$self` is present.
- Remote-style `$self` plus relative source URL produces the expected absolute URI.
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
- Add `ExpressionTypeObject`:
  - `type: "jsonpath" | "xpath" | "jsonpointer"`
  - optional `version`
  - defaults:
    - JSONPath: `rfc9535`
    - XPath: `xpath-31`
    - JSON Pointer: `rfc6901`
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
  - `$sourceDescriptions.<name>.<field-or-id>`
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
  - missing step ID
  - circular dependencies
  - dependency that never completes
- Keep `onSuccess`, `onFailure`, `goto`, `end`, and `retry` behavior compatible with current behavior.
- Clarify precedence between retry exhaustion and following failure actions.

Tests:
- Existing OpenAPI examples still run.
- Explicit `dependsOn` waits for required steps.
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
