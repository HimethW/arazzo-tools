# Arazzo v1.1.0 Support Plan

## Summary
Support Arazzo `1.1.0` across the full extension product: shared TypeScript models, Go LSP parser/validator/completions/navigation, visualizer, CLI/MCP runner, and tests, while preserving `1.0.0` and `1.0.1` compatibility.

Sources checked: [Arazzo v1.1.0 latest](https://spec.openapis.org/arazzo/latest.html) and [Arazzo v1.0.1](https://spec.openapis.org/arazzo/v1.0.1.html). Current repo is missing several v1.1.0 fields and behaviors: `$self`, `asyncapi`, `channelPath`, `timeout`, `correlationId`, `action`, step-level `dependsOn`, `querystring`, `Selector Object`, expanded `Expression Type Object`, `$message`, `$self`, richer source-description resolution, and newer selector/replacement semantics.

## Key Changes

- Update all Arazzo model definitions in TS and Go to accept `arazzo: "1.1.0"` plus the new root `$self` field.
- Extend `sourceDescriptions.type` from `openapi | arazzo` to `openapi | asyncapi | arazzo`.
- Add v1.1.0 step fields:
  `channelPath`, `timeout`, `correlationId`, `action: "send" | "receive"`, and `dependsOn`.
- Treat exactly one target selector as valid per step:
  `operationId`, `operationPath`, `channelPath`, or `workflowId`.
- Add `querystring` to parameter locations.
- Add `SelectorObject`:
  `context`, `selector`, and `type`.
- Add `ExpressionTypeObject`:
  `type: jsonpath | xpath | jsonpointer`, `version`, with defaults `rfc9535`, `xpath-31`, and `rfc6901`.
- Allow Selector Objects anywhere v1.1.0 permits them:
  workflow outputs, step outputs, parameter values, request body payload values, and payload replacement values.
- Extend payload replacements with:
  `targetSelectorType`, JSONPath targets, XPath targets, and Selector Object values.
- Keep existing non-spec extras such as `components.successCriteria` as backward-compatible tolerated fields, but do not advertise them as v1.1.0 fixed fields.

## Implementation Changes

- Centralize the Arazzo schema shape so `arazzo-designer-core`, LSP parser structs, CLI models, and visualizer data contracts do not drift again.
- Update LSP validation:
  accept `1.1.0`, validate `$self` as a URI-reference without fragment, allow `asyncapi`, validate new step fields, validate `querystring`, validate component key regexes, validate selectors/expression types, and reject invalid action/timeout/correlation combinations.
- Update LSP completions:
  default new files to `arazzo: "1.1.0"`, include `$self`, `asyncapi`, `channelPath`, `timeout`, `correlationId`, `action`, `dependsOn`, `querystring`, `targetSelectorType`, `$message`, and `$self`.
- Update navigation/indexing:
  continue OpenAPI `operationId`/`operationPath` navigation, add scoped `$sourceDescriptions.<name>.<id>` resolution, add AsyncAPI operation/channel indexing, and add navigation for `channelPath`.
- Implement v1.1.0 URI resolution:
  parse complete documents before resolving references, use `$self` as the document identity/base URI, resolve relative `sourceDescriptions.url` values from `$self` or retrieval URI, and match external Arazzo documents by `$self` when present.
- Update runtime expression evaluator:
  add `$message.header.*`, `$message.payload`, `$message.payload#/...`, `$self`, `$sourceDescriptions.<name>.<field-or-id>`, `$components.successActions.*`, `$components.failureActions.*`, and embedded expression serialization rules for objects/arrays.
- Replace the simple condition parser with a real expression evaluator for `!`, `&&`, `||`, grouping, indexing, property dereference, numeric comparison, and case-insensitive string comparison.
- Keep current RFC9535 JSONPath support, add selector evaluation as a shared service, and add XPath/XML support for criteria, selectors, and replacements.
- Update runner execution order:
  preserve sequential behavior when no dependencies exist, but honor explicit step `dependsOn` and implicit `$steps.<id>.outputs.*` dependencies before executing a step.
- Add AsyncAPI runtime as a pluggable adapter layer:
  core runner resolves AsyncAPI `operationId` or `channelPath`, evaluates `action`, `correlationId`, `timeout`, parameters, request body, and `$message`; broker/protocol-specific transport plugs in behind an adapter interface.
- For v1, ship the adapter interface plus an in-memory/test adapter and clear “no AsyncAPI adapter configured” runtime errors for real broker execution.
- Update visualizer:
  show `$self` and source type in overview, render AsyncAPI send/receive steps distinctly, show channel/correlation/timeout metadata in the properties panel, and draw dependency edges for explicit `dependsOn` relationships without breaking existing success/failure/goto layout.
- Update MCP/CLI workflow descriptions:
  expose v1.1.0 metadata, include async/source information in workflow details, and surface unsupported AsyncAPI adapter errors clearly to Copilot/MCP callers.

## Test Plan

- Add v1.1.0 fixture files covering every new field:
  `$self`, `asyncapi`, `channelPath`, `timeout`, `correlationId`, `action`, step `dependsOn`, `querystring`, Selector Objects, Expression Type Objects, `$message`, `$self`, and replacement `targetSelectorType`.
- Add compatibility tests proving existing `1.0.0` and `1.0.1` examples still parse, validate, visualize, and run.
- Add LSP tests for validation errors and completions for all new fields and enum values.
- Add runner tests for dependency ordering, implicit output dependencies, timeout failure, retry-after precedence, selector outputs, selector request bodies, JSONPath replacement, XPath replacement, and `$message` evaluation.
- Add AsyncAPI adapter tests using the in-memory adapter:
  send completes immediately, receive waits for correlated message, receive times out, and receive success criteria evaluate message payload.
- Add visualizer snapshot/graph tests for async steps and dependency edges.
- Run Go tests for CLI and LSP plus TypeScript build/tests for extension packages.

## Assumptions

- Scope is full product support: editor, validation, visualization, CLI runner, MCP execution, and tests.
- AsyncAPI execution will be pluggable, not hard-coded to a specific broker in this first v1.1.0 upgrade.
- Real broker adapters can be added after this plan without changing the Arazzo model or runner semantics.
- Existing Arazzo `1.0.0` and `1.0.1` behavior must remain backward compatible.
