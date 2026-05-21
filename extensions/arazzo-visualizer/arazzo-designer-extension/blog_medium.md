# From OpenAPI Endpoints to Runnable API Workflows: Introducing Arazzo Visualizer for VS Code

OpenAPI changed the way we describe APIs.

It gave teams a common language for documenting endpoints, request bodies, response schemas, authentication, examples, and everything else that makes an API understandable.

But most real API work does not happen one endpoint at a time.

A user signs in, gets a token, searches for a product, creates a cart, adds an item, checks out, and maybe confirms the order. A backend service calls one API, takes something from the response, passes it into the next request, branches on a condition, retries on failure, and validates that the whole flow worked.

That is where things usually get messy.

The API specification may be clean, but the actual workflow often ends up scattered across docs, test scripts, Postman collections, CI jobs, comments, and developer memory. You can read the endpoints, but you still have to mentally stitch together the journey.

That is the problem the [Arazzo Specification](https://spec.openapis.org/arazzo/latest.html) is designed to solve.

Arazzo is an OpenAPI Initiative specification for describing API workflows. Instead of only documenting individual operations, it lets you describe how multiple API calls work together: which step runs first, what data gets passed between steps, what success looks like, and how a real API journey behaves from start to finish.

I built **Arazzo Visualizer for VS Code** to make those workflows easier to create, understand, and run without leaving the editor.

It turns Arazzo files into interactive diagrams, adds smart editing support, connects workflows back to their OpenAPI operations, and includes a built-in runner so you can test real API sequences directly from VS Code.

![Arazzo Visualizer demo](https://raw.githubusercontent.com/wso2/arazzo-tools/main/extensions/arazzo-visualizer/arazzo-designer-extension/assets/v3_visualizer_demo.gif)

## Why Arazzo Matters Even More with AI

There is another reason Arazzo is becoming important: AI agents are starting to interact with APIs on our behalf.

Without a workflow description, an AI assistant has to figure out the API journey by itself. If you ask it to "create a cart and add a product," it may need to inspect the OpenAPI description, decide which endpoint to call first, understand how authentication works, identify which response fields to carry forward, choose the next endpoint, build each request, and recover when something fails.

That gives the AI a lot of freedom, but it also creates risk.

The flow can become inconsistent. The model may choose the wrong endpoint, miss a required dependency, pass the wrong value into a later request, or spend a lot of context reasoning through steps that your system already knows.

With Arazzo, the workflow is explicit.

The AI does not need to invent the flow every time. It can choose the right workflow, provide the required inputs, and let the runner execute the deterministic sequence defined in the Arazzo file.

That has a few practical benefits:

- **More control**: the workflow is designed by the API owner, not improvised at runtime
- **More repeatability**: the same task follows the same sequence each time
- **Less ambiguity**: dependencies, inputs, outputs, and success criteria are written down
- **Better safety**: the AI can operate through known workflows instead of freely exploring every endpoint
- **Potentially lower token usage**: the assistant can call a workflow tool instead of reasoning through and constructing every API call step by step

The token-saving part depends on the system design, but the direction is clear: when the workflow is already described, the AI needs less conversational context to rediscover it. The engine can handle the API calls, data passing, and validation, while the AI focuses on selecting the right workflow and interpreting the result.

That is a much better division of responsibility.

Let the specification define the workflow. Let the engine run it. Let the AI help the user choose, configure, and understand it.

## Why Arazzo Needs Good Tooling

Arazzo files are readable, but workflow files can still grow quickly.

Once you have multiple workflows, several steps, inputs, outputs, success criteria, conditional paths, and references into OpenAPI descriptions, it becomes easy to lose the shape of the flow.

You may find yourself asking:

- Which step depends on which?
- Where is this `operationId` defined?
- Is this response value used later?
- Did I break a step reference?
- Can I actually run this workflow against the real API?
- What failed when the workflow did not behave as expected?

Those are not problems a plain YAML editor can fully solve.

Arazzo Visualizer is meant to make Arazzo feel like a first-class development experience inside VS Code, not just another structured text file.

## The Basic Workflow

There are two common ways to use the extension.

The first path is when you already have OpenAPI descriptions and want to create an Arazzo workflow from them.

For example, you can open GitHub Copilot Chat and ask it to create an Arazzo file from your OpenAPI spec:

```text
Create a sample Arazzo file named toolshop.arazzo.yaml with 3 steps using the Toolshop OpenAPI specification below to list all products and create a cart:
https://api.practicesoftwaretesting.com/docs
```

Once Copilot creates the file, you can open it in VS Code, launch the visualizer, inspect the workflow, edit the steps, and run it.

The second path is when you already have an Arazzo file.

Open a `.arazzo.yaml`, `.arazzo.yml`, or `.arazzo.json` file, then click the **Arazzo Overview** icon in the editor toolbar or run:

```text
ArazzoDesigner: Open Arazzo Visualizer
```

The visualizer opens beside your file and stays in sync as you edit.

That means you can move between code and diagram naturally: change the YAML, see the diagram update, select a step, inspect the details, fix an issue, and run the workflow again.

## See the Workflow, Not Just the File

The main feature is the visual workflow diagram.

Instead of reading a long YAML or JSON file from top to bottom and building the flow in your head, you can see the workflow as connected steps.

You can:

- View the full workflow structure in one place
- Focus on a single workflow when a file contains several workflows
- Inspect request details, responses, inputs, outputs, and success criteria
- Understand dependencies between steps
- Watch the diagram update when the source file changes

This is especially useful when you are reviewing someone else's workflow, debugging a broken flow, or explaining an API journey to another developer.

Arazzo is about sequences. A diagram makes those sequences much easier to reason about.

## Smart Editing for Arazzo Files

The extension also improves the normal editing experience.

It recognizes Arazzo files such as:

- `.arazzo.yaml`
- `.arazzo.yml`
- `.arazzo.json`
- matching `-arazzo` file names

Once a file is recognized, you get language support for Arazzo-specific authoring:

- Syntax highlighting for Arazzo keywords
- Highlighting for runtime expressions like `$statusCode` and `$response.body`
- Suggestions for valid fields and values while typing
- Validation for missing fields, invalid references, and structure issues
- YAML and JSON support

This matters because small mistakes in workflow files can be painful. A broken `stepId`, an invalid reference, or a missing required field may not be obvious until much later.

The goal is to catch those issues while you are still writing.

## CodeLens Actions Where You Need Them

Arazzo Visualizer adds useful CodeLens actions directly above workflow definitions.

You do not need to remember commands or jump through menus. When you are looking at a workflow, the extension gives you actions in context.

The main actions are:

- **Visualize**: open the selected workflow in the visualizer
- **Try with curl**: run the workflow from the editor and see the result in a terminal
- **Try with AI**: hand the workflow to GitHub Copilot and run it through a natural language conversation

This keeps the workflow tight: write, visualize, run, inspect, adjust.

## Navigate from Arazzo Back to OpenAPI

An Arazzo workflow usually points back to operations defined in OpenAPI files.

That connection is important. If a workflow step says it uses an `operationId`, you often want to jump straight to the operation definition and confirm the method, path, request body, response schema, or summary.

Arazzo Visualizer includes language-server support for this.

You can use **Ctrl+Click** on Windows/Linux or **Cmd+Click** on macOS to navigate from an Arazzo `operationId` to the matching OpenAPI operation.

You can also hover over an `operationId` to see operation details without leaving the file, including information such as the HTTP method, path, summary, and where the operation is defined.

The extension can discover OpenAPI files near your Arazzo file, including files in the same directory, nearby subdirectories, and a parent directory. It also re-indexes when OpenAPI files change, so navigation stays useful as the API evolves.

For teams with multiple OpenAPI files, this saves a lot of context switching.

## Create and Edit Workflows with GitHub Copilot

One of the most useful parts of the extension is the Copilot workflow.

Arazzo is expressive, but writing a workflow from scratch can still feel unfamiliar if you are new to the spec. Copilot can help you get the first draft in place.

You can ask Copilot to create an Arazzo workflow from an OpenAPI description, then use the visualizer to review and refine it.

For example:

```text
Create an Arazzo workflow that authenticates a user, lists available products, creates a cart, and adds the first product to the cart.
```

You can also ask Copilot to modify an existing workflow:

```text
Add success criteria to the create-cart step to check that the status code is 200.
```

```text
Add a retry step if the product list request fails.
```

After saving the file, the visualizer updates to match the latest version.

This gives you a practical loop: use AI to draft or adjust the workflow, use the visual diagram to understand it, and use validation plus execution to verify it.

## Run Workflows from VS Code

Documentation is useful. Runnable documentation is better.

Arazzo Visualizer includes a built-in runner engine, so you can execute workflows directly from VS Code without setting up a separate tool.

You can run API calls in the order defined by the Arazzo file, pass data from one step into later steps, validate success criteria, and inspect what happened during the run.

![Arazzo workflow execution demo](https://raw.githubusercontent.com/wso2/arazzo-tools/main/extensions/arazzo-visualizer/arazzo-designer-extension/assets/v3_execution_demo.gif)

The runner helps answer the question that matters most:

Does this workflow actually work against the API?

When a workflow runs, the extension can show:

- Live execution progress
- Which steps passed
- Which steps failed
- Response and status information
- Output values passed between steps
- Trace details for troubleshooting

This makes Arazzo useful not only as documentation, but also as a development and testing asset.

## Try with curl

Sometimes you do not want an AI-assisted flow. You just want to run the workflow directly and see the result.

That is what **Try with curl** is for.

You can trigger it from CodeLens above a workflow or from the visualizer. The extension builds the curl-based execution flow, opens the terminal output, and animates the execution path in the diagram while the workflow runs.

![Try with curl demo](https://raw.githubusercontent.com/wso2/arazzo-tools/main/extensions/arazzo-visualizer/arazzo-designer-extension/assets/v3_curl_demo.gif)

If the workflow has inputs, the extension opens an input configuration panel before running. Required fields are clearly marked, and the panel prevents the run from starting until required values are filled.

This is useful for workflows that need things like usernames, IDs, tokens, environment-specific values, or other runtime parameters.

## Better Handling for Real Development Environments

Real APIs are not always clean demo environments.

Sometimes you are testing against local services, internal systems, staging deployments, or endpoints with self-signed certificates.

If a run fails because of TLS certificate validation, Arazzo Visualizer detects the certificate error and offers a shortcut to disable TLS validation for the workspace. You can also manage this manually through VS Code settings:

```text
Settings -> Extensions -> Arazzo Visualizer
```

There is also a server control button in the editor toolbar while the Arazzo server is running, so you can stop it without leaving the editor.

These are small details, but they matter in day-to-day use. Tooling should not fall apart the moment you leave the happy path.

## Execution Logs and Tracing

When a workflow fails, you need more than a red icon.

The extension includes execution logs and trace details so you can understand what happened step by step.

You can review workflow runs, inspect failures, look at request and response details, and identify where the flow slowed down or broke.

This is especially helpful for multi-step API journeys where the real problem may be several calls earlier than the failing step.

For example, a checkout step may fail because the cart ID was not extracted correctly from a previous response. A visual workflow plus execution logs makes that easier to find.

## A Small Look Under the Hood

This post is mainly about what the extension does, but I think the architecture is worth a quick mention.

Arazzo Visualizer is built as a VS Code extension with a few cooperating parts.

The language server handles the pro-code editing experience: validation, completions, diagnostics, CodeLens actions, hover information, and navigation between Arazzo workflows and OpenAPI operation definitions.

The visualizer runs in a VS Code webview. It communicates with the extension through an RPC layer, keeping the diagram, editor state, workflow actions, and execution state connected.

The runner integration makes it possible to execute workflows from the editor, the diagram, curl-based actions, or Copilot-assisted flows. Copilot can also start the Arazzo server, run workflows, and update settings such as TLS validation when needed.

I plan to write a separate technical deep dive on the architecture, including the LSP, webview, RPC layer, runner, and Copilot integration. For this launch post, the important part is simpler: the extension brings authoring, visualization, navigation, and execution into one VS Code workflow.

## Who This Is For

Arazzo Visualizer is useful if you:

- Work with OpenAPI and want to describe real API journeys
- Need to document multi-step API workflows
- Want to test end-to-end API flows from your editor
- Are exploring the Arazzo Specification
- Build developer tools, SDKs, or API automation around OpenAPI
- Want Copilot to help create and run API workflows
- Prefer visual feedback when working with complex YAML or JSON files

If you are already using OpenAPI, Arazzo is a natural next step when endpoint-level documentation is not enough.

And if you are trying Arazzo for the first time, the extension is meant to make that first experience much less abstract.

## Try It

You can install **Arazzo Visualizer** from the VS Code Marketplace:

[Arazzo Visualizer on the VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=WSO2.arazzo-visualizer)

The source code is available on GitHub:

[wso2/arazzo-tools](https://github.com/wso2/arazzo-tools)

To get started:

1. Install the extension.
2. Open or create an `.arazzo.yaml`, `.arazzo.yml`, or `.arazzo.json` file.
3. Click the **Arazzo Overview** icon or run **ArazzoDesigner: Open Arazzo Visualizer**.
4. Visualize the workflow.
5. Try running it with curl or Copilot.

If it helps your workflow, a star on GitHub is always appreciated. If you find a bug or have an idea for a feature, open an issue.

Arazzo gives us a standard way to describe API workflows. My goal with Arazzo Visualizer is to make those workflows easier to build, easier to understand, and easier to prove in the place many developers already live: VS Code.
