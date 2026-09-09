# Phase 12 / step 2 — Failure labels (`error_class`)

When a workflow fails, the runner sends back a **sentence**. A person reads it fine; a program
cannot. Its only option was to go hunting for words inside our prose —

```js
if (error.includes("timed out")) { retry(); }     // breaks the day anyone rewords the message
```

— which is fragile, and those messages *do* get reworded.

So a failure now carries a short **label** beside the message. Same idea as HTTP: the body says
*"Sorry, we couldn't find that page"* and the status line says `404`. People read the body, programs
read the code. This adds the code.

```json
{
  "status": "failed",
  "error_class": "adapter_unsupported",
  "error": "the \"kafka\" protocol is not yet supported: a Kafka adapter … is a planned future phase"
}
```

The message is **unchanged**. The label sits next to it.

## The ten labels

Defined in one place — [internal/failure/failure.go](../../../../arazzo-designer-cli/internal/failure/failure.go).
Adding, renaming or removing one happens there and nowhere else.

**Every runtime failure the runner can produce has one.** That completeness is the point: an empty
`error_class` no longer means "this situation has no name", it means a code path was added without
one — which a test now catches at the line it was written.

| label | what happened | retry? | example |
|---|---|---|---|
| `adapter_unsupported` | the protocol has no adapter (kafka, amqp, …) | **never** | 01 |
| `connect_failed` | the remote endpoint could not be reached — broker or HTTP | **yes** — commonly transient | 02 |
| `receive_timeout` | nothing arrived before the timeout | **maybe** — with a longer timeout | 03 |
| `correlation_unresolved` | the `correlationId` expression produced no value | no — fix the document | 04 |
| `serialize_failed` | the message could not be encoded or decoded | no — fix the document | 05 |
| `target_unresolved` | points at something that does not exist | no — fix the **referenced spec** | 06 |
| `document_invalid` | the document itself is malformed | no — fix **this file** | 07 |
| `criteria_unmet` | it all worked and the answer was wrong | no — retrying reproduces it | 08 |
| `dependency_unmet` | the step never ran; a prerequisite had not completed | not on its own | 09 |
| `unsupported_feature` | valid per spec, not implemented yet | never | 10 |

That retry column is the whole point. Examples 01 and 02 both look identical from the outside — a
send that failed — and the correct response to them is opposite.

Three pairs are deliberately kept apart because they send you to different places:

- **06 vs 07** — `target_unresolved` sends you to the *referenced* spec; `document_invalid` sends you
  to *this* file.
- **07 vs 10** — `document_invalid` means you wrote it wrong; `unsupported_feature` means the file is
  correct and the runner has not built it yet. Telling someone to "fix" a correct document sends them
  chasing a problem that is not there.
- **08 vs everything else** — `criteria_unmet` is a product regression; the rest are infrastructure.
  For a CI pipeline that is the most valuable line in the table.

**Labels are attached where the failure is created**, never worked out afterwards from the message
text. That is what makes them survive rewording.

## Scenarios

One example per label, so the set is a complete tour of the vocabulary.

| file | workflow | expected `error_class` |
|---|---|---|
| `01-adapter-unsupported.arazzo.yaml` | `clickStream` | `adapter_unsupported` |
| `02-connect-failed.arazzo.yaml` | `deadBroker` | `connect_failed` |
| `03-receive-timeout.arazzo.yaml` | `waitForever` | `receive_timeout` |
| `04-correlation-unresolved.arazzo.yaml` | `badCorrelation` | `correlation_unresolved` |
| `05-serialize-failed.arazzo.yaml` | `avroSend` | `serialize_failed` |
| `06-target-unresolved.arazzo.yaml` | `brokenTarget` | `target_unresolved` |
| `07-document-invalid.arazzo.yaml` | `noDirection` | `document_invalid` |
| `08-criteria-unmet.arazzo.yaml` | `wrongAnswer` | `criteria_unmet` |
| `09-dependency-unmet.arazzo.yaml` | `blocked` | `dependency_unmet` |
| `10-unsupported-feature.arazzo.yaml` | `crossDoc` | `unsupported_feature` |
| `11-success.arazzo.yaml` | `roundTrip` | *(succeeds — no error keys)* |

**Ten of the eleven are meant to fail.** That failure is the expected result, not a problem with your
setup. Run `11` first to confirm the setup works at all.

**Nothing here needs the internet.** Kafka and the dead broker fail before any connection is made or
at `127.0.0.1:1`, which refuses instantly; everything else runs on the in-memory adapter.

## How to test

Unlike step 1, this needs **no MCP handshake** — `/run` is a plain HTTP endpoint. One curl per
example.

The IDE route is first; the by-hand terminal route follows it.

### In the development environment (VS Code) — the recommended way

The extension already has a one-click runner for this, and it hits the same `/run` endpoint.

1. Launch the **Extension Development Host** and open one of the `.arazzo.yaml` files above.
2. A **`▶ Try with curl`** CodeLens sits directly above each `workflowId`. Click it.
   - If the server is not running for this file, it asks *"The Arazzo server is not currently running
     for this file. Start it to run the workflow?"* → **Yes**. It picks a port itself; you never need
     to know it. (First launch may raise a Windows **firewall prompt** — private networks is enough.)
   - It opens the visualizer for that workflow first, which is normal.
3. A terminal named **Arazzo** appears with the command **typed but not run**. Press **Enter**.
4. Read the output. On Windows the command ends in `| Format-List`, so you get one property per line:

   ```
   status      : failed
   error_class : adapter_unsupported
   error       : the "kafka" protocol is not yet supported: a Kafka adapter …
   ```

   For **example 11** — the only one that succeeds — there is no `error` or `error_class` line at
   all. Every other example prints one.
5. **`Stop Arazzo Server`** from the Command Palette when you are done.

> **Edited the file?** The CodeLens changes to **`▶ Retry`** and prompts you to restart the server —
> it serves the document as it was when it started, so an edit needs a restart.

**One server serves one document**, so switching examples means letting it restart when it asks.

#### Running example 04 both ways in the IDE

`04` is the one worth doing twice. Its `token` input is declared but optional, so:

- **Click `▶ Try with curl` and press Enter** → no token → `correlation_unresolved`
- **Set the input, then run again** → `receive_timeout`

To set it, use **Configure Inputs** in the visualizer panel for that workflow and give `token` any
value (e.g. `abc`). Or just edit the command in the terminal before pressing Enter — change
`'{"inputs":{}}'` to `'{"inputs":{"token":"abc"}}'`.

Same workflow, two different labels. That is the clearest demonstration in the set.

#### Through Copilot instead

With the server running the extension writes `.vscode/mcp.json`, so Copilot can call the workflow as
an MCP tool. Ask it to run e.g. `clickStream`. On that path the label is embedded in the message:

```
Workflow failed [adapter_unsupported]: the "kafka" protocol is not yet supported: …
```

MCP reports a tool failure as text rather than a structured body, which is why it is written inline
there. Be aware Copilot may summarise rather than quote — if you are checking an exact label, ask for
the raw JSON.

### Straight from a terminal

Same endpoint, driven by hand.

#### Start the server

```bash
arazzo-designer-cli serve -f examples/async_test/phase12/step2_errorLabels/01-adapter-unsupported.arazzo.yaml -p 8791
```

> Windows may raise a **firewall prompt** the first time — the server opens a listening socket.
> Allowing it on *private* networks is enough, and localhost keeps working even if you decline.
> `Ctrl+C` stops it. One document per server, so restart it when you switch examples.

#### Run the workflow

```bash
curl -s -X POST http://localhost:8791/run/clickStream -H 'Content-Type: application/json' -d '{}'
```

That's the whole test. The response body is the thing to look at.

## What to expect

Verified output, `POST /run/{workflowId}` with `-d '{}'`. **Every one returns HTTP 200** — the
request succeeded; the workflow didn't. That is deliberate, and it is why the class lives in the body
rather than in the status code.

### 01 → `adapter_unsupported`

```json
{ "status": "failed",
  "error_class": "adapter_unsupported",
  "error": "the \"kafka\" protocol is not yet supported: a Kafka adapter (with Avro/Protobuf schema support) is a planned future phase - supported protocols: ws, wss, mqtt, mqtts (and in-memory when no servers are declared)" }
```

### 02 → `connect_failed`

```json
{ "status": "failed",
  "error_class": "connect_failed",
  "error": "send on channel \"orders\" failed: websocket connect to ws://127.0.0.1:1/orders failed: dial tcp 127.0.0.1:1: … actively refused it." }
```

**Compare this with 01.** Both are "a send that failed". One is worth retrying and one never will be,
and the only thing in the response that tells them apart is the label.

### 03 → `receive_timeout`

```json
{ "status": "failed",
  "error_class": "receive_timeout",
  "error": "receive on channel \"local/events\" timed out after 500ms: no message arrived" }
```

### 04 → `correlation_unresolved`, and the same workflow → `receive_timeout`

```json
{ "status": "failed",
  "error_class": "correlation_unresolved",
  "error": "receive on channel \"local/events\": correlationId \"$inputs.token\" resolved to no value, so there is no id to match - refusing to fall back to an unfiltered receive" }
```

Now run **the same workflow** with the input supplied:

```bash
curl -s -X POST http://localhost:8791/run/badCorrelation -H 'Content-Type: application/json' -d '{"inputs":{"token":"abc"}}'
```

```json
{ "status": "failed",
  "error_class": "receive_timeout",
  "error": "receive on channel \"local/events\" timed out after 500ms: no message matching correlationId \"abc\" arrived" }
```

The correlation now resolves, so the step gets as far as waiting — and fails differently. **Two
classes from one workflow**, which is the clearest demonstration that the label tracks what actually
happened rather than which workflow was run.

### 05 → `serialize_failed`

```json
{ "status": "failed",
  "error_class": "serialize_failed",
  "error": "send on channel \"records\": could not serialize payload: avro serialization (application/avro) is not supported yet" }
```

The runner *recognises* `application/avro` — it is a deliberate stub awaiting schema-registry
support — so selection succeeds and encoding fails. Reporting "unsupported content type" at
selection time would have been a lie.

### 06 → `target_unresolved`

```json
{ "status": "failed",
  "error_class": "target_unresolved",
  "error": "AsyncAPI target could not be resolved (channel or operation not found)" }
```

`localBus` is a real declared source; that AsyncAPI file just has no channel called
`nosuchchannel`. The reference is well-formed, the thing it names is missing.

> The editor does **not** flag this today. The LSP checks that a `channelPath` names a declared
> source of type `asyncapi`, but never that the channel exists inside it — so this reaches the
> runtime. Worth fixing in the LSP; the lookup it needs already exists.

### 07 → `document_invalid`

```json
{ "status": "failed",
  "error_class": "document_invalid",
  "error": "a 'channelPath' step requires 'action' (send or receive) - the message-flow direction is otherwise undefined" }
```

**Compare with 06.** There the channel was missing; here it exists and is perfectly reachable — the
step just never says which way the message goes. Different file to open, hence a different label.

### 08 → `criteria_unmet`

```json
{ "status": "failed",
  "error_class": "criteria_unmet",
  "error": "received message did not satisfy successCriteria" }
```

**Nothing technically failed.** The message was sent, it arrived, it decoded — the assertion on it
did not hold. This is an ordinary test failure, and the most common one a CI run sees. Without its
own label, "the API returned the wrong value" and "the broker was down" arrive looking identical, and
a pipeline cannot separate a product regression from flaky infrastructure.

### 09 → `dependency_unmet`

```json
{ "status": "failed",
  "error_class": "dependency_unmet",
  "error": "Dependency execution failed: … step 'needsLater' dependsOn 'later', which has not completed successfully" }
```

`dependsOn` is a **gate, not a scheduler** — it does not reorder steps and does not trigger them. So
`needsLater`, declared first, runs first and finds its prerequisite has not succeeded.

This is the only label where **the step never ran at all**. Its own correctness is still unknown, and
the failure to chase belongs to the prerequisite. A report that shows it as a normal failure misleads.

### 10 → `unsupported_feature`

```json
{ "status": "failed",
  "error_class": "unsupported_feature",
  "error": "step 'needsRemote' dependsOn '$sourceDescriptions.other.wf.steps.s': cross-document step dependencies are not yet supported" }
```

The document is **correct**. Cross-document `dependsOn` is valid per the spec; the runner has not
implemented it. That is why it is not `document_invalid` — telling someone to fix a correct file
sends them chasing a problem that is not there.

### 11 → success

```json
{ "status": "success",
  "outputs": { "seen": "ok-1" } }
```

No `error`, no `error_class`. Classification changed nothing for a workflow that works — which is the
regression this example exists to catch.

## Notes

- **HTTP status codes were deliberately not reused** for these. They are already doing a different
  job here: `400` for bad inputs, `404` for an unknown workflow, `200` for "the workflow ran, here is
  the outcome". Putting HTTP codes in the body too would make the same numbers mean two things at two
  levels. And `receive_timeout` has no HTTP equivalent at all — `504` means an upstream never answered
  a *request*, and nothing sent one.
- **`GET /lastResult/{workflowId}`** returns the cached response of the most recent run, label
  included, without executing anything again.
- **Every runtime failure path is labelled**, and a test enforces it by reading the source rather
  than running it: any `createFailureResult` call or failure result literal that omits a class fails
  the build with the file and line. A behaviour test can only cover the paths someone thought to
  provoke; this catches a new one on the day it is written.
- The field can still say **"I don't know"** — an empty class remains meaningful and is pinned by a
  test — but nothing in the runtime produces one today. If you ever see one, it is a missing label,
  not a nameless situation.
- **Labels are additive, never renamed.** Adding a class is safe: a client ignores what it does not
  recognise. Renaming or merging one breaks every client silently, because a comparison against the
  old string simply stops matching and reports nothing. That is why the grouping was settled before
  any of this shipped.
