# chief-summarizer

A Go CLI tool that automatically generates structured markdown summaries for all `.md` files in a directory tree using local Ollama models.

## Overview

`chief-summarizer` walks through your markdown documents, intelligently chunks them, and creates comprehensive summaries using a local Ollama instance. Each summary is saved alongside the original file with a `_summary.md` suffix.

## Features

### Core Functionality
- 🔄 **Batch Processing**: Automatically processes all markdown files in a directory tree
- 📝 **Smart Chunking**: Character-based text splitting with configurable size and overlap
- 🤖 **Ollama Integration**: Uses local Ollama models (qwen2.5:14b, llama3.1:8b, mistral:7b)
- 🎯 **Intelligent Model Selection**: Automatic fallback to closest available model variant
- 📊 **Progress Tracking**: Live chunk and merge indicators during processing
- 🎲 **Randomized Processing**: Processes files in random order each run to avoid bias
- ⚙️ **Highly Configurable**: Extensive CLI flags for customization
- 🔄 **Automatic Updates**: Self-updates from GitHub releases on each execution

### Requirements
- Go 1.21 or later
- Running Ollama instance (default: `http://localhost:11434`)
- Config file at `~/.config/chiefsummarizer.yaml` (optional)

## Usage

### Basic Invocation
```bash
chief-summarizer [flags] [rootPath]
```

The `rootPath` can be either:
- A directory tree (processes all `.md` files recursively)
- A single markdown file
- Omitted if `processing.root_path` is set in the config file

**Note**: If no config file exists at `~/.config/chiefsummarizer.yaml`, all settings use their defaults and `rootPath` must be provided as a command line argument.

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-host` | `http://localhost:11434` | Ollama server URL |
| `-model` | auto-detect | Override model selection |
| `-chunk-size` | `4000` | Characters per chunk |
| `-chunk-overlap` | `400` | Overlap between chunks |
| `-force` | `false` | Overwrite existing summaries |
| `-dry-run` | `false` | Show what would be done |
| `-max-files` | unlimited | Maximum files to process |
| `-verbose` | `false` | Detailed output |
| `-quiet` | `false` | Minimal output |
| `-exclude` | none | Regex pattern to exclude files (repeatable) |
| `-request-timeout` | `10m` | HTTP request timeout |
| `-disable-autoupdate` | `false` | Disable automatic update checks |
| `-version` | - | Show version info |

### Output Status Codes
- `OK`: Successfully processed
- `SKIP`: Skipped (summary exists, not forced)
- `DRY`: Dry-run mode (no action taken)
- `ERR`: Error occurred

### Exit Codes
- `0`: All files processed successfully
- `1`: One or more errors occurred

## How It Works

`chief-summarizer` follows a systematic pipeline:

1. **Update Check**
   - On each execution, checks GitHub for newer releases
   - Automatically downloads and updates the binary if a new version is available
   - Continues with normal operation after update

2. **Initialization**
   - Parse CLI flags and validate `rootPath`
   - Negotiate model selection (override → auto-detect → fallback)
   - Configure HTTP timeout

3. **File Discovery**
   - Walk directory tree using `filepath.WalkDir`
   - Select `.md` files that don't end in `_summary.md`
   - Apply exclusion patterns (`-exclude`)
   - Shuffle file list for randomized processing

4. **Processing Pipeline**
   - Read markdown document
   - Split into chunks at rune boundaries (respects size/overlap)
   - Generate chunk summaries via Ollama API
   - Categorize document length (SHORT/MEDIUM/LONG)
   - Hierarchically consolidate chunk summaries (max 4 inputs per merge)
   - Synthesize final summary from consolidated chunks
   - Write `<name>_summary.md` atomically

5. **Error Handling**
   - Continue processing on recoverable errors
   - Track error state per file
   - Exit with code `1` if any errors occurred

## Prompt Templates

### Chunk Summary Prompt
```
You are "Chief Summarizer", an assistant that creates concise summaries in the original language of the text.

Task:
- Read the following markdown excerpt.
- Write a short summary of this excerpt.
- Use the SAME LANGUAGE as the text (usually German).
- Keep names, dates and key facts accurate.
- Do NOT add your own interpretations or new ideas.
- Do NOT write an overall document summary, only summarize THIS excerpt.

Output format:
- 1 short paragraph in plain text (no headings).
- Maximum ~120 words.

Excerpt:
---
{{CHUNK_CONTENT}}
---
```

### Intermediate Merge Prompt
```
You are "Chief Summarizer", an assistant that consolidates partial summaries into a concise overview while keeping the original language.

Task:
- Merge the following partial summaries from the same document into a single partial summary.
- Remove duplicated information and resolve conflicts.
- Maintain the SAME LANGUAGE as the inputs (usually German).
- Keep important names, dates and numbers.
- Use 1–2 short paragraphs OR 3–5 bullet points.
- Do NOT add headings, intro text, or any sections labelled "Thinking".

Input partial summaries:
---
{{PARTIAL_SUMMARIES}}
---

Return ONLY the consolidated partial summary, nothing else.
```

### Final Combined Summary Prompt
```
You are "Chief Summarizer", an assistant that creates structured summaries in the original language of the source text.

Task:
- You receive several partial summaries of different excerpts of ONE long markdown document.
- Combine them into ONE cohesive summary.
- Remove repetition and contradictions.
- Maintain the SAME LANGUAGE as the original text (usually German).
- Keep important names, dates and numbers.
- Be neutral and factual.
- Do **not** output any "Thinking" paragraphs or hidden reasoning traces.

Output format (proper Markdown with headings):

1. Start with a level-2 heading: ## Ultra-Kurzfassung
2. Below it, write two short sentences:
   - Line 1: one short sentence describing the main topic.
   - Line 2: one short sentence describing the main outcome or conclusion.

3. Then add a blank line.

4. Then add another level-2 heading: ## Ausführliche Zusammenfassung
5. Below it, write the detailed summary:
   - If the original document was short (~< 1.500 Wörter):
     - write 2–4 short paragraphs OR 3–6 bullet points.
   - If the original document was medium (1.500–5.000 Wörter):
     - write 3–6 paragraphs and optionally 3–8 bullet points.
   - If the original document was long (> 5.000 Wörter):
     - use clear markdown headings (### level-3) and bullet lists for structure.
   - Always stay focused on the key points, decisions, arguments, and results.

IMPORTANT: Use proper markdown headings (## and ###) throughout. The output must be valid markdown.

Input:
Original document length category: {{CATEGORY}}

The following are partial summaries of the document, in order:

---
{{CHUNK_SUMMARIES}}
---

Now produce ONLY the markdown summary as specified above.
Do not add any intro text or explanations around it.
```

## Installation

### Prerequisites
1. Install [Go](https://golang.org/dl/) 1.21 or later
2. Install and start [Ollama](https://ollama.ai/)
3. Pull a supported model:
   ```bash
   ollama pull qwen2.5:14b
   # or
   ollama pull llama3.1:8b
   # or
   ollama pull mistral:7b
   ```
4. (Optional) Create and configure config file:
   ```bash
   mkdir -p ~/.config
   cp chiefsummarizer.yaml.example ~/.config/chiefsummarizer.yaml
   # Edit the file to set your preferences
   nano ~/.config/chiefsummarizer.yaml
   ```
   
   The config file is optional. If it doesn't exist, all settings use their defaults. CLI flags always take precedence over config file values.

### Build & Install

```bash
# Build the binary
make build

# Install to ~/.local/bin
make install

# Clean build artifacts
make clean
```

Ensure `~/.local/bin` is on your `PATH`:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Automation with systemd

Automate summary generation with the provided systemd units (runs every 2 hours, processes max 3 files).

### Setup

1. **Copy systemd units**
   ```bash
   mkdir -p ~/.config/systemd/user
   cp systemd/chief-summarizer.* ~/.config/systemd/user/
   ```

2. **Configure the service**
   Edit `~/.config/systemd/user/chief-summarizer.service` and update the `ExecStart` path:
   ```ini
   ExecStart=%h/.local/bin/chief-summarizer --max-files 3 /path/to/your/documents
   ```
   Replace `/path/to/your/documents` with your actual document directory.

3. **Verify prerequisites**
   - Binary installed: `which chief-summarizer`
   - PATH includes `~/.local/bin`
   - (Optional) Config exists: `ls ~/.config/chiefsummarizer.yaml`

4. **Enable and start**
   ```bash
   systemctl --user daemon-reload
   systemctl --user enable --now chief-summarizer.timer
   ```

5. **Check status**
   ```bash
   systemctl --user status chief-summarizer.timer
   systemctl --user list-timers
   ```

### Timer Configuration
- **Interval**: Every 2 hours (`OnUnitActiveSec=2h`)
- **Limit**: Max 3 files per run (`--max-files 3`)
- **Type**: User-level service (no root required)

## Project Structure

```
.
├── cmd/
│   └── chief-summarizer/
│       └── main.go          # Main CLI entry point
├── systemd/
│   ├── chief-summarizer.service
│   └── chief-summarizer.timer
├── Makefile
└── README.md
```

### Key Components
- **`main.go`**: CLI flag parsing, model selection, file discovery, and processing pipeline
- **`Config`**: Holds all CLI configuration options
- **`processFile`**: Core pipeline for reading, chunking, summarizing, and writing
- **Systemd units**: User-level automation for scheduled summarization

## Automatic Updates

`chief-summarizer` includes self-update functionality powered by [go-github-selfupdate](https://github.com/rhysd/go-github-selfupdate):

- **Update Check**: On each execution, the tool checks GitHub for newer releases
- **Automatic Download**: If a new version is available, it downloads and replaces the binary automatically
- **Version Display**: Use `-version` flag to see the current version
- **Disable Updates**: Use `-disable-autoupdate` flag or set `updates.disable_autoupdate: true` in config file

### For Maintainers: Creating Releases

To enable automatic updates, releases must include binary assets. Follow this complete workflow:

#### Step 1: Update Version
Edit `cmd/chief-summarizer/main.go` and update the version constant:
```go
const version = "1.0.3"
```

Commit the change:
```bash
git add cmd/chief-summarizer/main.go
git commit -m "Bump version to 1.0.3"
git push
```

#### Step 2: Create and Push Tag
```bash
git tag -a v1.0.3 -m "Release version 1.0.3"
git push origin v1.0.3
```

#### Step 3: Cross-Compile Binaries
Build for all supported platforms:

```bash
# Linux AMD64 (most common Linux systems)
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o chief-summarizer-linux-amd64 ./cmd/chief-summarizer

# Linux ARM64 (Raspberry Pi, ARM servers)
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o chief-summarizer-linux-arm64 ./cmd/chief-summarizer

# macOS AMD64 (Intel Macs)
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o chief-summarizer-darwin-amd64 ./cmd/chief-summarizer

# macOS ARM64 (Apple Silicon Macs - M1/M2/M3)
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o chief-summarizer-darwin-arm64 ./cmd/chief-summarizer

# Windows AMD64 (optional)
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o chief-summarizer-windows-amd64.exe ./cmd/chief-summarizer
```

**Build flags explained:**
- `-ldflags="-s -w"`: Strip debug info and reduce binary size
- `GOOS`: Target operating system
- `GOARCH`: Target architecture

#### Step 4: Create GitHub Release

**Option A: Using GitHub Web Interface**

1. Go to your repository: https://github.com/danst0/Chief-Summarizer
2. Click "Releases" → "Draft a new release"
3. Choose the tag `v1.0.3` (the one you just pushed)
4. Set release title: `v1.0.3`
5. Add release notes describing changes
6. Drag and drop all compiled binaries to the assets section:
   - `chief-summarizer-linux-amd64`
   - `chief-summarizer-linux-arm64`
   - `chief-summarizer-darwin-amd64`
   - `chief-summarizer-darwin-arm64`
   - `chief-summarizer-windows-amd64.exe` (if included)
7. Click "Publish release"

**Option B: Using GitHub CLI (`gh`)**

```bash
# Install gh CLI if not available: https://cli.github.com/

# Create release and upload binaries
**Option A: Using GitHub CLI (Recommended)**

```bash
gh release create v1.0.3 \
   --title "v1.0.3" \
  --notes "Release notes here" \
  chief-summarizer-linux-amd64 \
  chief-summarizer-linux-arm64 \
  chief-summarizer-darwin-amd64 \
  chief-summarizer-darwin-arm64 \
  chief-summarizer-windows-amd64.exe
```

#### Step 5: Verify Auto-Update

Test that the self-update mechanism works:
```bash
# Users on older versions will see:
chief-summarizer -version
# Output: "New version 1.0.3 is available! (current: 1.0.0)"
#         "Updating binary..."
#         "Successfully updated to version 1.0.3"
```

#### Important Notes

- **Binary naming**: The self-update library automatically detects the correct binary based on OS/architecture
- **File permissions**: Make binaries executable after download (library handles this automatically)
- **Semantic versioning**: Always use proper semver format (v1.0.0, v1.0.3, etc.)
- **Release notes**: Document breaking changes, new features, and bug fixes

#### Build Script (Optional)

Create a `scripts/build-release.sh` for convenience:

```bash
#!/bin/bash
VERSION=${1:-"dev"}

echo "Building Chief Summarizer v${VERSION} for all platforms..."

# Create dist directory
mkdir -p dist

# Build flags
LDFLAGS="-s -w"

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/chief-summarizer-linux-amd64 ./cmd/chief-summarizer
GOOS=linux GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o dist/chief-summarizer-linux-arm64 ./cmd/chief-summarizer

# macOS
GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/chief-summarizer-darwin-amd64 ./cmd/chief-summarizer
GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o dist/chief-summarizer-darwin-arm64 ./cmd/chief-summarizer

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/chief-summarizer-windows-amd64.exe ./cmd/chief-summarizer

echo "✓ Binaries built in dist/"
ls -lh dist/
```

Usage:
```bash
chmod +x scripts/build-release.sh
./scripts/build-release.sh 1.0.3
```

The self-update mechanism will automatically detect and apply updates based on semantic versioning.

## Contributing

Contributions are welcome! Future improvements:
- Modular packages (`internal/chunking`, `internal/ollama`, `internal/prompts`)
- Enhanced error handling and recovery
- Support for additional LLM backends
- Parallel processing optimizations
- Automated CI/CD pipeline for releases

## License

MIT License - see LICENSE file for details.
