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

## The five labels

Defined in one place — [internal/failure/failure.go](../../../../arazzo-designer-cli/internal/failure/failure.go).
Adding, renaming or removing one happens there and nowhere else.

| label | what happened | retry? |
|---|---|---|
| `adapter_unsupported` | the protocol has no adapter (kafka, amqp, …) | **never** |
| `connect_failed` | the broker could not be reached | **yes** — commonly transient |
| `receive_timeout` | nothing arrived before the timeout | **maybe** — with a longer timeout |
| `correlation_unresolved` | the `correlationId` expression produced no value | no — fix the document |
| `serialize_failed` | the message could not be encoded or decoded | no — fix the document |

That retry column is the whole point. Examples 01 and 02 both look identical from the outside — a
send that failed — and the correct response to them is opposite.

**Labels are attached where the failure is created**, never worked out afterwards from the message
text. That is what makes them survive rewording, and it is why example 06 exists.

## Scenarios

| file | workflow | expected `error_class` |
|---|---|---|
| `01-adapter-unsupported.arazzo.yaml` | `clickStream` | `adapter_unsupported` |
| `02-connect-failed.arazzo.yaml` | `deadBroker` | `connect_failed` |
| `03-receive-timeout.arazzo.yaml` | `waitForever` | `receive_timeout` |
| `04-correlation-unresolved.arazzo.yaml` | `badCorrelation` | `correlation_unresolved` |
| `05-serialize-failed.arazzo.yaml` | `avroSend` | `serialize_failed` |
| `06-unclassified-failure.arazzo.yaml` | `brokenTarget` | **no key at all** |
| `07-success.arazzo.yaml` | `roundTrip` | *(succeeds — no error keys)* |

**Six of the seven are meant to fail.** That failure is the expected result, not a problem with your
setup. Run `07` first to confirm the setup works at all.

**Nothing here needs the internet.** Kafka and the dead broker fail before any connection is made or
at `127.0.0.1:1`, which refuses instantly; everything else runs on the in-memory adapter.

## How to test

Unlike step 1, this needs **no MCP handshake** — `/run` is a plain HTTP endpoint. One curl per
example.

### Start the server

```bash
arazzo-designer-cli serve -f examples/async_test/phase12/step2_errorLabels/01-adapter-unsupported.arazzo.yaml -p 8791
```

> Windows may raise a **firewall prompt** the first time — the server opens a listening socket.
> Allowing it on *private* networks is enough, and localhost keeps working even if you decline.
> `Ctrl+C` stops it. One document per server, so restart it when you switch examples.

### Run the workflow

```bash
curl -s -X POST http://localhost:8791/run/clickStream -H 'Content-Type: application/json' -d '{}'
```

That's the whole test. The response body is the thing to look at.

### From VS Code instead

1. Open the `.arazzo.yaml` and run **`Start Arazzo Server`** — note the port in the notification.
2. Use the **`▶ Try with curl`** CodeLens above the workflow, or ask Copilot to run the workflow.
   Via the MCP tool the label appears in the message itself: `Workflow failed [adapter_unsupported]: …`
3. **`Stop Arazzo Server`** when you are done.

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

### 06 → no label at all

```json
{ "status": "failed",
  "error": "AsyncAPI target could not be resolved (channel or operation not found)" }
```

**Check the raw JSON, not a formatted summary.** There is no `error_class` key — not empty, *absent*.

This is the honesty check. A wrong-but-plausible label is worse than none, because a client will act
on it. "I don't know" has to be expressible, and absence is how it is expressed. A pretty-printer or
an LLM summary may quietly omit a missing key, so look at the body itself.

### 07 → success

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
- A failure outside the five is left unlabelled rather than forced into the nearest fit. Most
  non-async failures — a missing operation, a failed success criterion — are in that group today.
