/**
 * Builds the Arazzo Language Server for all platforms shipped in the VSIX.
 * Cross-platform equivalent of arazzo-designer-lsp/build.sh — works on Windows without bash/WSL.
 */
const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const lspProjectDir = path.join(__dirname, '..', '..', 'arazzo-designer-lsp');
const outputDir = path.join(__dirname, '..', 'ls');

const targets = [
    ['darwin', 'arm64', 'arazzo-language-server-darwin-arm64'],
    ['darwin', 'amd64', 'arazzo-language-server-darwin-amd64'],
    ['linux', 'amd64', 'arazzo-language-server-linux-amd64'],
    ['linux', 'arm64', 'arazzo-language-server-linux-arm64'],
    ['windows', 'amd64', 'arazzo-language-server.exe']
];

if (!fs.existsSync(path.join(lspProjectDir, 'go.mod'))) {
    console.error(`Arazzo Language Server project not found: ${lspProjectDir}`);
    process.exit(1);
}

// Clean and recreate output directory
fs.rmSync(outputDir, { recursive: true, force: true });
fs.mkdirSync(outputDir, { recursive: true });

for (const [goos, goarch, outputName] of targets) {
    const outputPath = path.join(outputDir, outputName);
    console.log(`Building Arazzo Language Server for ${goos}/${goarch} -> ${outputName}`);

    const result = spawnSync('go', ['build', '-o', outputPath, 'main.go'], {
        cwd: lspProjectDir,
        env: {
            ...process.env,
            CGO_ENABLED: '0',
            GOOS: goos,
            GOARCH: goarch
        },
        stdio: 'inherit'
    });

    if (result.error) {
        console.error(`Failed to spawn go build: ${result.error.message}`);
        process.exit(1);
    }

    if (result.status !== 0) {
        process.exit(result.status ?? 1);
    }

    if (goos !== 'windows') {
        fs.chmodSync(outputPath, 0o755);
    }
}

console.log(`Arazzo Language Server binaries are ready at: ${outputDir}`);
