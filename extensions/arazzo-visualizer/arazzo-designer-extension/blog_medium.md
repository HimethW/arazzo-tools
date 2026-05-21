# From OpenAPI Endpoints to Runnable API Workflows: Introducing Arazzo Visualizer for VS Code

## Quick Overview

OpenAPI is great at describing individual API endpoints, but many real API tasks are workflows, not single calls. A user signs in, receives a token, searches for data, creates something, passes an ID into the next request, checks a result, and continues through a sequence. Those multi-step journeys often end up hidden in documentation, test scripts, Postman collections, CI jobs, or developer memory.

The [Arazzo Specification](https://spec.openapis.org/arazzo/latest.html) gives teams a standard way to describe those API workflows. It defines how multiple API operations work together: what runs first, which inputs are required, how data from one step is used in another step, what success looks like, and how the whole flow should behave.

This matters because API consumers usually need to understand the journey, not just the endpoint list. Instead of asking every developer, tester, tool, or AI assistant to rediscover the correct sequence of calls, Arazzo lets the API owner write the workflow down once in a structured, repeatable, and testable format.

**Arazzo Visualizer for VS Code** is an extension that makes those workflow files easier to work with inside the editor. It turns Arazzo files into interactive diagrams, provides editing support for YAML and JSON workflow files, and lets you run workflows directly from VS Code.

In practice, the extension helps you:

- Open an overview page for the whole Arazzo file
- Understand API workflows visually instead of reading long YAML files line by line
- Create and refine Arazzo workflows with GitHub Copilot
- Navigate from workflow steps to the related OpenAPI operations
- Catch authoring mistakes with validation, completions, hovers, and diagnostics
- Execute workflows with curl or Copilot-assisted flows
- Inspect execution progress, outputs, failures, and traces

If you are still reading, the rest of this post walks through why Arazzo matters, how the extension works, and how you can use it in real API development.

## In This Post

- [Why Arazzo Matters](#why-arazzo-matters)
- [Using the VS Code Extension](#using-the-vs-code-extension)
- [A Small Look Under the Hood](#a-small-look-under-the-hood)
- [Who This Is For](#who-this-is-for)
- [Try It](#try-it)

## Why Arazzo Matters

### AI Needs Reliable Workflows

AI agents are starting to interact with APIs on our behalf, which makes explicit workflows even more valuable.

Without a workflow description, an assistant has to infer the API journey by itself: which endpoint to call first, how authentication works, which response fields to carry forward, what to do next, and how to recover from failure. That freedom is useful, but it can also lead to inconsistent flows, missed dependencies, wrong values, and extra reasoning over steps your system already knows.

With Arazzo, the workflow is explicit. The AI can choose the right workflow, provide the required inputs, and let the runner execute the deterministic sequence defined in the Arazzo file.

That has a few practical benefits:

- **More control**: the workflow is designed by the API owner, not improvised at runtime
- **More repeatability**: the same task follows the same sequence each time
- **Less ambiguity**: dependencies, inputs, outputs, and success criteria are written down
- **Better safety**: the AI can operate through known workflows instead of freely exploring every endpoint
- **Potentially lower token usage**: the assistant can call a workflow tool instead of reasoning through and constructing every API call step by step

The token-saving part depends on the system design, but the direction is clear: when the workflow is already described, the AI needs less context to rediscover it. The engine can handle API calls, data passing, and validation while the AI focuses on selecting the workflow.

### Why Arazzo Needs Good Tooling

Arazzo files are readable, but they can grow quickly. With multiple workflows, several steps, inputs, outputs, success criteria, conditional paths, and OpenAPI references, it becomes easy to lose the shape of the flow.

You may find yourself asking:

- Which step depends on which?
- Where is this `operationId` defined?
- Is this response value used later?
- Did I break a step reference?
- Can I actually run this workflow against the real API?
- What failed when the workflow did not behave as expected?

Those are not problems a plain YAML editor can fully solve. That is where dedicated Arazzo tooling starts to matter.

## Using the VS Code Extension

![Arazzo Visualizer demo](https://raw.githubusercontent.com/wso2/arazzo-tools/main/extensions/arazzo-visualizer/arazzo-designer-extension/assets/v3_visualizer_demo.gif)

This section follows the normal user flow: create or open an Arazzo file, inspect the whole file in the overview page, drill into a workflow, start the runner, execute the workflow, and use traces or logs when something fails.

### 1. Start with an Arazzo File

If you already have an Arazzo file, open a `.arazzo.yaml`, `.arazzo.yml`, or `.arazzo.json` file in VS Code.

If you do not have one yet, you can ask GitHub Copilot Chat to create a first draft from an OpenAPI description:

```text
Create a sample Arazzo file named toolshop.arazzo.yaml with 3 steps using the Toolshop OpenAPI specification below to list all products and create a cart:
https://api.practicesoftwaretesting.com/docs
```

Once the file exists, the extension recognizes Arazzo-specific file names such as:

- `.arazzo.yaml`
- `.arazzo.yml`
- `.arazzo.json`
- matching `-arazzo` file names

From there, the editor gives you syntax highlighting, runtime-expression highlighting for values like `$statusCode` and `$response.body`, completions, validation, diagnostics, and YAML or JSON support. That catches broken `stepId`s, invalid references, missing fields, and structure issues while you are still writing.

### 2. Open the Arazzo Overview

With the Arazzo file open, click the **Arazzo Overview** button in the editor title bar.

> Image placeholder: Arazzo Overview button in the VS Code editor title bar.

You can also open it from the command palette:

```text
ArazzoDesigner: Open Arazzo Visualizer
```

The overview page gives you a file-level view of the Arazzo document. Instead of jumping straight into a workflow graph, you can first inspect the document title, version, description, source descriptions, workflows, and reusable components.

The workflows are listed as cards. Each card shows the workflow ID, summary, and number of steps. Click a workflow card to open the visual workflow view.

If you already know which workflow you want, you can also use the **Visualize** CodeLens directly above a workflow definition in the editor. That opens the workflow view without going through the overview page.

### 3. Inspect a Workflow Visually

The workflow view shows the selected workflow as connected steps. This is where the Arazzo file becomes easier to reason about, especially when the workflow has several dependencies.

You can:

- View the workflow structure as a graph
- Click a step to inspect its properties
- Review request details, responses, inputs, outputs, and success criteria
- Understand which steps depend on earlier outputs
- Move back to the overview page from the workflow view

The visualizer stays connected to the source file, so you can move between the YAML or JSON and the diagram while you refine the workflow.

### 4. Navigate Back to OpenAPI

An Arazzo workflow usually points back to operations defined in OpenAPI files. If a step uses an `operationId`, you often want to confirm the method, path, request body, response schema, or summary.

You can use **Ctrl+Click** on Windows/Linux or **Cmd+Click** on macOS to navigate from an Arazzo `operationId` to the matching OpenAPI operation.

You can also hover over an `operationId` to see operation details without leaving the file, including the HTTP method, path, summary, and where the operation is defined.

The extension discovers OpenAPI files near your Arazzo file, including files in the same directory, nearby subdirectories, and a parent directory. It also re-indexes when OpenAPI files change.

### 5. Start the Arazzo Server

When you are ready to execute workflows, click the **play** button in the editor title bar.

That starts the bundled Arazzo server for the active file. Behind the scenes, the extension:

- Starts the Arazzo runner
- Registers the workflows as MCP tools for VS Code by writing `.vscode/mcp.json`
- Starts a local trace server so workflow execution can be mapped back onto the diagram
- Shows server output in the VS Code terminal and trace events in the Output panel

Once the server is running, two more CodeLens actions appear above each workflow in the editor:

- **Try with curl**
- **Try with AI**

The same actions are available from the workflow webview, so you can run the workflow from either the editor or the diagram.

There is also a stop button while the server is running. Use it to stop the Arazzo server, stop trace collection, and remove the run-specific CodeLens actions when you are done.

### 6. Configure Inputs When Needed

Some workflows need starting inputs such as usernames, IDs, tokens, or environment-specific values. In the workflow view, click the gear icon to open the input configuration panel.

> Image placeholder: Configure Inputs panel opened from the gear icon.

Required fields are clearly marked. The extension prevents a curl run from starting until required values are filled. Saved inputs are reused for future runs of the same workflow, and the extension cross-checks them against the current file so stale or renamed fields are not sent by mistake.

### 7. Run with curl

**Try with curl** is useful when you want a direct, deterministic execution path.

When you click it, the extension generates a request to the runner endpoint for that workflow. On Windows it uses `Invoke-RestMethod`; on macOS and Linux it uses `curl`. The command is placed in the VS Code terminal so you can review or edit it before running.

![Try with curl demo](https://raw.githubusercontent.com/wso2/arazzo-tools/main/extensions/arazzo-visualizer/arazzo-designer-extension/assets/v3_curl_demo.gif)

During execution, the graph highlights the path the workflow is taking. Steps show live status, passed or failed state, response information, output values, and timing details.

### 8. Run with AI

**Try with AI** opens GitHub Copilot with a workflow-specific prompt. Because the play button registered the Arazzo server as an MCP server, Copilot can discover the available workflow tools, choose the right one, provide the required inputs, and ask the runner to execute the workflow.

![Arazzo workflow execution demo](https://raw.githubusercontent.com/wso2/arazzo-tools/main/extensions/arazzo-visualizer/arazzo-designer-extension/assets/v3_execution_demo.gif)

This keeps the AI from having to invent the API sequence on its own. The workflow remains the source of truth, while Copilot helps select and run it.

Copilot can also help create or modify workflows before you run them. For example:

```text
Create an Arazzo workflow that authenticates a user, lists available products, creates a cart, and adds the first product to the cart.
```

Or:

```text
Add success criteria to the create-cart step to check that the status code is 200.
```

After saving the file, the visualizer updates so you can review the change immediately.

### 9. Read Logs and Traces

When a workflow fails, you need to see more than a red icon.

Each completed step can show a **view logs** link in the graph. Click it to open the logs for that step and inspect the execution status, request and response details, outputs, errors, and timing.

You can also inspect traces outside the graph. The trace server exposes a JSON endpoint:

```text
http://127.0.0.1:59600/api/traces
```

If port `59600` is already in use, the trace server moves to the next available port. The exact trace server port is printed in the task/output logs, and you can open the endpoint in a browser or call it from a tool such as Postman.

The generated command and runner result are visible in the VS Code terminal while the workflow runs. In the Output panel, you can also select **Arazzo Trace Server** to inspect raw trace events.

### 10. Handle TLS Issues in Local or Staging Environments

Real APIs are not always clean demo environments. Sometimes you are testing against local services, internal systems, staging deployments, or endpoints with self-signed certificates.

If a run fails because of TLS certificate validation, Arazzo Visualizer detects the certificate error and offers a shortcut to disable TLS validation for the workspace. You can also manage this manually through VS Code settings:

```text
Settings -> Extensions -> Arazzo Visualizer
```

The setting is called `arazzo.disableTLSCertificationValidation`. The Arazzo server must be restarted for the change to take effect.

## A Small Look Under the Hood

Arazzo Visualizer is built as a VS Code extension with a few cooperating parts.

The language server handles the pro-code editing experience: validation, completions, diagnostics, CodeLens actions, hover information, and navigation between Arazzo workflows and OpenAPI operation definitions.

The visualizer runs in a VS Code webview and communicates with the extension through an RPC layer, keeping the diagram, editor state, workflow actions, and execution state connected.

The runner integration executes workflows from the editor, the diagram, curl-based actions, or Copilot-assisted flows. Copilot can also start the Arazzo server, run workflows, and update settings such as TLS validation when needed.

I plan to write a separate technical deep dive on the LSP, webview, RPC layer, runner, and Copilot integration. For this launch post, the important part is simpler: the extension brings authoring, visualization, navigation, and execution into one VS Code workflow.

## Who This Is For

Arazzo Visualizer is useful if you:

- Work with OpenAPI and want to describe real API journeys
- Need to document multi-step API workflows
- Want to test end-to-end API flows from your editor
- Are exploring the Arazzo Specification
- Build developer tools, SDKs, or API automation around OpenAPI
- Want Copilot to help create and run API workflows
- Prefer visual feedback when working with complex YAML or JSON files

If you are already using OpenAPI, Arazzo is a natural next step when endpoint-level documentation is not enough. If you are trying Arazzo for the first time, the extension is meant to make that first experience much less abstract.

## Try It

You can install **Arazzo Visualizer** from the VS Code Marketplace:

[Arazzo Visualizer on the VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=WSO2.arazzo-visualizer)

The source code is available on GitHub:

[wso2/arazzo-tools](https://github.com/wso2/arazzo-tools)

If it helps your workflow, a star on GitHub is always appreciated. If you find a bug or have an idea for a feature, open an issue.

Arazzo gives us a standard way to describe API workflows. My goal with Arazzo Visualizer is to make those workflows easier to build, easier to understand, and easier to prove in the place many developers already live: VS Code.
