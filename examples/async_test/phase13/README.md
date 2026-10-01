# Phase 13 — graph: step icons, `dependsOn` on hover, and red dashed for steps that never ran

Three graph changes. The hover and the red dash have three examples each; the icons show on every graph. All use real endpoints: the **Toolshop API**
(`api.practicesoftwaretesting.com`) for REST steps and the **public WebSocket echo server**
(`echo.websocket.org`) for async steps. **Internet required. No inputs needed** — click Run.

## What the colours mean

| on the node | meaning |
|---|---|
| green border | ran and passed |
| red **solid** border | ran and failed |
| red **dashed** border | never ran — a `dependsOn` prerequisite had not succeeded (new) |
| no colour | not reached |
| yellow-orange border, while hovering another step | a prerequisite of the step you are hovering (new) |

## Step icons (new)

Each step shows the logo for what it targets, drawn in the theme's text colour:

| icon | the step targets |
|---|---|
| OpenAPI logo | an OpenAPI operation |
| AsyncAPI logo | an AsyncAPI operation or channel |
| Arazzo logo | another workflow — in this document or another Arazzo document |
| arrow (the old icon) | anything whose type cannot be worked out |

The icon always matches the *Step Type* field in the properties panel — both come from the same code.
Examples 01, 02, 04 and 05 show OpenAPI steps; 03 and 06 show AsyncAPI steps.

**All three on one graph:** open
[`../phase12/step1_workflowInfo/05-mixed-steps.arazzo.yaml`](../phase12/step1_workflowInfo/05-mixed-steps.arazzo.yaml),
workflow `orderThenWait` — no run needed. `placeOrder` shows OpenAPI, `alsoEmit` and `waitForOrder`
show AsyncAPI, `handOff` shows Arazzo. `placeOrder` and `alsoEmit` are written identically (a bare
`operationId`), so their icons differ only because the language server looked each one up in the
declared specs.

## Before you start

The extension loads its **own copies** of the graph bundle and the runner, so rebuild both before
launching the Extension Development Host:

```bash
pnpm --dir extensions/arazzo-visualizer/arazzo-designer-visualizer run build
pnpm --dir extensions/arazzo-visualizer/arazzo-designer-extension run build-cli
```

The icons and the hover need only the first command. The red dash needs both — it is the runner that now reports
a step that never ran.

## Hover scenarios — hover works before a run, and after one

| file | workflow | hover | lights up | run result |
|---|---|---|---|---|
| `01-hover-one-dependency` | `oneDependency` | `listProducts` | `listBrands` only — not `listCategories`, which sits in between | ✅ all green |
| `02-hover-several-and-chained` | `severalAndChained` | `listProducts` | `listBrands` **and** `listCategories` | ✅ all green, `productName = "Combination Pliers"` |
| | | `productDetail` | `listProducts` only — direct prerequisites, not the chain behind them | |
| `03-hover-async-cross-workflow` | `echoRoundTrip` | `hear` | `shout` only — its other prerequisite is in the `warmUp` workflow, not in this graph | ✅ all green, `echoed = "hello from phase 13"` |

Hovering a step with no `dependsOn` changes nothing. After a run the highlight shows over the
green/red border for as long as you hover.

## Red-dash scenarios — each ends in error on purpose

| file | workflow | what you see |
|---|---|---|
| `04-dash-prerequisite-failed` | `prerequisiteFailed` | `findBrand` **solid** red (404) → `listBrandProducts` **dashed** red |
| `05-dash-prerequisite-skipped` | `prerequisiteSkipped` | `listBrands` green → `listCategories` no colour (jumped over) → `listProducts` **dashed** red |
| `06-dash-async-timeout` | `asyncPrerequisiteTimedOut` | `shout` green → `awaitReply` **solid** red (3s timeout) → `confirm` **dashed** red → `wrapUp` no colour |

In each, *view logs* on the dashed step gives the reason, e.g.
`step 'listProducts' dependsOn 'listCategories', which has not completed successfully`, and the
workflow ends with `error_class: dependency_unmet`. Hover the dashed step to light up the
prerequisite that stopped it.

What each shows:
- **04** — the two reds side by side: one step broke, the other was stopped by it.
- **05** — dashed with **no** solid red: nothing failed, a `goto` skipped the prerequisite.
- **06** — the same with async steps, and a refused step **ends the run** (`wrapUp` is never reached).

## Expected editor diagnostics

None on 01, 02, 04, 05. On 03 and 06 there is one blue **information** hint on the receive step
(*no correlation id location is declared in source 'echoServer'*). It is about the echo server's
spec — the Phase 11 echo examples show the same one — and is not part of Phase 13.

## From the command line (runner only — no graph)

```bash
cd arazzo-designer-cli
go run ./test_runner ../examples/async_test/phase13/01-hover-one-dependency.arazzo.yaml oneDependency
go run ./test_runner ../examples/async_test/phase13/02-hover-several-and-chained.arazzo.yaml severalAndChained
go run ./test_runner ../examples/async_test/phase13/03-hover-async-cross-workflow.arazzo.yaml echoRoundTrip
go run ./test_runner ../examples/async_test/phase13/04-dash-prerequisite-failed.arazzo.yaml prerequisiteFailed
go run ./test_runner ../examples/async_test/phase13/05-dash-prerequisite-skipped.arazzo.yaml prerequisiteSkipped
go run ./test_runner ../examples/async_test/phase13/06-dash-async-timeout.arazzo.yaml asyncPrerequisiteTimedOut
```

01–03 end `workflow_complete`; 04–06 end `error` with `error_class: dependency_unmet`. For 04–06
`test_runner` also prints `FAILED` / `SOME TESTS FAILED` — it treats any error as a failure, but here
the error is the expected outcome.
