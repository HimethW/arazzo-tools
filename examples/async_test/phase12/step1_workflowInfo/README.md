# Phase 12 / step 1 — Workflow information (`get_workflow_details`)

Phase 12 surfaces what phases 8–11 built. Nothing new *runs* here; the runner is unchanged. What
changed is what the tool can **tell you about a workflow before running it**.

`GetWorkflowDetails` ([runner.go](../../../../arazzo-designer-cli/internal/runner/runner.go)) was
written when every step was a REST call, and reported exactly five keys per step — `stepId`,
`operationId`, `operationPath`, `workflowId`, `description`. For an event-driven step that meant five
nulls: the fields that make it async were never read, and the facts that matter most aren't written
in the Arazzo file at all.

It now reports three groups of things.

| group | keys | where it comes from |
|---|---|---|
| **document** | `arazzoVersion`, `self`, `sourceDescriptions[]` | the Arazzo file |
| **step, as written** | `channelPath`, `action`, `correlationId`, `timeout`, `dependsOn` | the Arazzo file |
| **step, resolved** | `stepType`, `channel`, `action`, `adapter` / `adapterError`, `contentTypes`, `correlationIdLocations` | resolving the step into the referenced **AsyncAPI document** |

The last group is the point. It resolves through the *same* helpers the executor uses
(`ResolveAsyncTarget`, `TransportForSource`), never re-derived from the step text — so a description
cannot disagree with what a run would do. A step whose bare `operationId` resolves to an AsyncAPI
operation is reported as async even though it has no `channelPath` anywhere.

**Everything is additive.** No existing key changed or disappeared; new keys appear only when they
have a value, so a REST step describes exactly as it always did.

## Scenarios

| file | what it tests | workflow | the thing to look at |
|---|---|---|---|
| `01-rest-only.arazzo.yaml` | backwards compatibility | `browse` | both steps have **only** the five original keys + `stepType: openapi` — no async keys at all |
| `02-targeting-forms.arazzo.yaml` | all three ways to target a channel | `threeWays` | three differently-written steps resolve to the **same** channel, adapter and declarations |
| `03-unsupported-protocol.arazzo.yaml` | a protocol with no adapter | `clickStream` | `adapterError` explains why it cannot run, **without contacting anything** |
| `04-inmemory.arazzo.yaml` | no `servers` section | `localOnly` | `adapter: in-memory` — a named choice, not a blank |
| `05-mixed-steps.arazzo.yaml` | all three step types together | `orderThenWait` | `openapi`, `asyncapi` and `workflow` side by side in one step list |
| `06-ambiguous-declarations.arazzo.yaml` | a channel whose messages disagree | `notify` | **two** content types and **two** correlation locations, not one of each |

Each file's header comment states its full expected output. Supporting specs: `catalog.openapi.yaml`,
`orders.asyncapi.yaml` (mqtt), `stream.asyncapi.yaml` (kafka), `local.asyncapi.yaml` (no servers),
`mixedformats.asyncapi.yaml` (wss, two message formats).

## How to run these

`get_workflow_details` is an **MCP tool**, and the MCP server is its only caller. There is no CLI
subcommand for it, so you reach it through the server.

### In the development environment (VS Code)

1. Open one of the `.arazzo.yaml` files above in the Extension Development Host.
2. Run **`Start Arazzo Server`** from the Command Palette (`arazzo.startMCPServer`).
   - It picks a **random port in 18080–19079** and shows it in the notification.
   - Windows may raise a **firewall prompt** the first time — the server opens a listening socket.
     Allowing it on *private* networks is enough, and localhost keeps working even if you decline.
   - It also writes `.vscode/mcp.json`, which is what lets Copilot see the tool.
3. Click **`Try Now`** on the notification (or open Copilot Chat yourself) and ask:

   > get the workflow details for threeWays

   Copilot calls `get_workflow_details` and shows the JSON.
4. Run **`Stop Arazzo Server`** when you are done — it keeps listening otherwise.

### Straight from a terminal

Same server, no editor. Start it on a port you choose:

```bash
arazzo-designer-cli serve -f examples/async_test/phase12/step1_workflowInfo/02-targeting-forms.arazzo.yaml -p 8791
```

Then do the MCP handshake — `SID` is the `Mcp-Session-Id` response header of the `initialize` call:

```bash
curl -sD - -X POST http://localhost:8791/mcp -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"cli","version":"1"}}}'
```

and call the tool:

```bash
curl -s -X POST http://localhost:8791/mcp -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' -H "Mcp-Session-Id: $SID" -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_workflow_details","arguments":{"workflow_id":"threeWays"}}}'
```

Stop the server with `Ctrl+C` when you are done.

## What to expect

### 01 — `browse` (REST only)

Document: `arazzoVersion: "1.1.0"`, `self: "./01-rest-only.arazzo.yaml"`, one source `catalog/openapi`.
Both steps look like:

```json
{ "stepId": "listAll", "operationId": "getProducts", "operationPath": null,
  "workflowId": null, "description": "…", "stepType": "openapi" }
```

**The check here is what is absent.** No `channel`, `action`, `adapter`, `adapterError`,
`contentTypes`, `correlationIdLocations`, `channelPath`, `correlationId` or `timeout`. A v1.1.0
document costs a REST step nothing.

### 02 — `threeWays` (three targeting forms)

All three steps report the **same** three values:

```
channel                 arazzo/phase12/orders      ← the broker TOPIC, not the channel key "orders"
adapter                 mqtt
contentTypes            ["application/json"]
correlationIdLocations  ["$message.payload#/token"]
```

and differ only in `action`: `send`, **`receive`**, `send`.

Two things worth noticing:

- `viaOperationId` names **no source description at all** (`operationId: consumeOrder`). Reading the
  Arazzo file cannot tell you what it targets — only looking inside the declared spec can. It still
  comes back fully resolved.
- its `action` is `receive` even though the step never says so: the **operation's** action wins over
  the step's, exactly as it does at run time.

### 03 — `clickStream` (kafka)

```
stepType asyncapi   channel click-events   action send   contentTypes ["application/json"]
```

There is no `adapter` key. Instead:

```
adapterError  the "kafka" protocol is not yet supported: a Kafka adapter (with Avro/Protobuf
              schema support) is a planned future phase - supported protocols: ws, wss, mqtt,
              mqtts (and in-memory when no servers are declared)
```

Run the same workflow and that is the error, character for character. Describing gets you the answer
with no broker, no connection and no wait.

### 04 — `localOnly` (no servers)

```
stepType asyncapi   channel local/events   action send   adapter in-memory
contentTypes ["application/json"]
```

and **no `correlationIdLocations`** — the document declares none. That absence is meaningful: it is
exactly when a receive cannot filter on a declared location and falls back to scanning the whole
message.

### 05 — `orderThenWait` (all three step types)

| step | `stepType` | notable |
|---|---|---|
| `placeOrder` | `openapi` | a **bare** `operationId` in a **two-source** document, resolved to the OpenAPI one; gains no async keys |
| `waitForOrder` | `asyncapi` | `channel arazzo/phase12/orders`, `action receive`, `adapter mqtt`, `correlationId $inputs.token`, `timeout 5000`, `dependsOn ["placeOrder"]` |
| `handOff` | `workflow` | `workflowId: localOnly`, and none of the async fields |

`placeOrder` and example 02's `viaOperationId` are written identically — a bare `operationId` — and
resolve to different types. That difference is only visible by looking inside the specs.

### 06 — `notify` (ambiguous channel)

```
stepType asyncapi   channel notices   action send   adapter websocket
contentTypes            ["application/json", "text/plain"]
correlationIdLocations  ["$message.header#/traceId", "$message.payload#/token"]
```

Two entries each, sorted. The channel carries two messages that disagree, and the document cannot say
which one a given step will carry — so both are reported rather than one being picked. A single value
would have looked settled when it is not.

## Notes

- **Describing never connects.** `orders.asyncapi.yaml` names `broker.hivemq.com` and
  `mixedformats.asyncapi.yaml` names `echo.websocket.org`, but `get_workflow_details` resolves the
  transport from the document's `servers` declaration without building an adapter. No internet is
  needed for any example here.
- `channel` is the **address** (the broker-side topic), deliberately different from the channel *key*
  used in `channelPath` — resolving is what turns one into the other. Example 02 shows `orders`
  becoming `arazzo/phase12/orders`.
- `action` is handled twice on purpose: passed through from the step, then overwritten by the
  operation's action when the step resolves to one. The reported value is always the one that will
  actually be used.
- `list_workflows` and the run tools are **not** enriched yet — that is step 2. This step is
  `get_workflow_details` alone.
