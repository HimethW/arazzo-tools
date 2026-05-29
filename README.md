# arazzo-tools
A collection of developer tools, VS Code extensions, utilities, and examples for the OpenAPI Initiative’s Arazzo Specification, enabling real-time workflow execution, visualization, validation, and an improved developer experience for API orchestration.

## Development Setup

### 1. Prerequisites
- **Node.js**: Windows (v20.x or v22.x LTS)
- **Go**: Required for CLI and LSP development
- **Rush**: Install globally via `npm install -g @microsoft/rush`

### 2. Environment Configuration
Rush manages a project-specific version of `pnpm` (currently v10.11.0). To build the project, you must add the Rush-installed `pnpm` to your `PATH` for your current terminal session:

```powershell
# PowerShell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
$env:PATH = "$env:USERPROFILE\.rush\node-v22.15.0\pnpm-10.11.0\node_modules\.bin;" + $env:PATH
```

### 3. Build Instructions
From the root directory:

```powershell
# Install dependencies
rush install

# Build the extension and all internal dependencies
rush build -t arazzo-visualizer
```

### 4. Running/Debugging the Extension
1. **Start the Visualizer UI dev server**:
   ```powershell
   cd extensions/arazzo-visualizer/arazzo-designer-visualizer
   npm run start
   ```
2. **Launch VS Code Extension**:
   - Return to the main project folder in VS Code.
   - Press `F5` (or click **Run and Debug** > **Run Extension**).
   - A new [Extension Development Host] window will open.


If the watcher keeps failing due to PowerShell execution policy restrictions, run:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

After debugging, restore the previous state with:

```powershell
Set-ExecutionPolicy Undefined -Scope CurrentUser
```
