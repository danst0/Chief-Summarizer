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

### Requirements
- Go 1.21 or later
- Running Ollama instance (default: `http://localhost:11434`)
- Config file at `~/.config/chiefsummarizer.yaml` (required)

## Usage

### Basic Invocation
```bash
chief-summarizer [flags] <rootPath>
```

The `rootPath` can be either:
- A directory tree (processes all `.md` files recursively)
- A single markdown file

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

1. **Initialization**
   - Parse CLI flags and validate `rootPath`
   - Negotiate model selection (override → auto-detect → fallback)
   - Configure HTTP timeout

2. **File Discovery**
   - Walk directory tree using `filepath.WalkDir`
   - Select `.md` files that don't end in `_summary.md`
   - Apply exclusion patterns (`-exclude`)
   - Shuffle file list for randomized processing

3. **Processing Pipeline**
   - Read markdown document
   - Split into chunks at rune boundaries (respects size/overlap)
   - Generate chunk summaries via Ollama API
   - Categorize document length (SHORT/MEDIUM/LONG)
   - Synthesize final summary from chunks
   - Write `<name>_summary.md` atomically

4. **Error Handling**
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

Output format (Markdown, fixed):

1. First, write a very short two-line "Ultra-Kurzfassung" overview:
   - Line 1: one short sentence describing the main topic.
   - Line 2: one short sentence describing the main outcome or conclusion.

2. Then add a blank line.

3. Then write the "Ausführliche Zusammenfassung" (detailed summary) in markdown:
   - If the original document was short (~< 1.500 Wörter):
     - write 2–4 short paragraphs OR 3–6 bullet points.
   - If the original document was medium (1.500–5.000 Wörter):
     - write 3–6 paragraphs and optionally 3–8 bullet points.
   - If the original document was long (> 5.000 Wörter):
     - use clear markdown headings (##) and bullet lists for structure.
   - Always stay focused on the key points, decisions, arguments, and results.

Input:
The following are partial summaries of the document, in order:

---
{{CHUNK_SUMMARIES}}
---

Now produce ONLY the markdown summary as specified above.
Do not add any intro text or explanations around it.
```

> **Tip**: Prepend `Original document length category: SHORT|MEDIUM|LONG.` before the `Input:` block so the model can adjust the layout accordingly.

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
4. Create config file:
   ```bash
   mkdir -p ~/.config
   cp chiefsummarizer.yaml.example ~/.config/chiefsummarizer.yaml
   ```
   
   The config file can be empty for now (all settings are via CLI flags), but it must exist.

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
   - Config exists: `ls ~/.config/chiefsummarizer.yaml`
   - PATH includes `~/.local/bin`

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

## Contributing

Contributions are welcome! Future improvements:
- Modular packages (`internal/chunking`, `internal/ollama`, `internal/prompts`)
- Enhanced error handling and recovery
- Support for additional LLM backends
- Parallel processing optimizations

## License

MIT License - see LICENSE file for details.
