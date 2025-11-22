# chief-summarizer

`chief-summarizer` is a Go CLI that batch-generates structured markdown summaries for every `.md` document inside a directory tree. It orchestrates chunk-level and document-level calls to a local Ollama instance, writing the final result next to the source file.

## 1. Feature & CLI specification
- **Invocation**: `chief-summarizer [flags] <rootPath>` where `rootPath` must be supplied and can reference either a directory tree or a single markdown file.
- **File discovery**: walk every subdirectory, select regular `.md` files that do not already end in `_summary.md`.
- **Summary placement**: write `<name>_summary.md` beside each source; skip when the file already exists unless `-force` is used.
- **Chunking**: character-based slices with configurable size/overlap via `-chunk-size` (default 4000) and `-chunk-overlap` (default 400).
- **Model selection**: connect to `GET /api/tags` on the Ollama host (default `http://localhost:11434`) and pick the best available model from `qwen2.5:14b`, `llama3.1:8b`, `mistral:7b`, unless `-model` overrides it. If the exact model is missing, the tool automatically picks the closest installed variant (matching base names or prefixes) before falling back to any available model.
- **Flags**: include `-host`, `-model`, `-chunk-size`, `-chunk-overlap`, `-force`, `-dry-run`, `-max-files`, `-verbose`, `-quiet`, `-exclude <regex>` (repeatable), `-request-timeout <duration>` (default `5m`), `-version`.
- **Console output**: one status line per candidate (`OK`, `SKIP`, `DRY`, `ERR`) showing file paths, chunk counts, selected model, and failure reasons where applicable, plus live `CHNK` / `MERGE` indicators while Ollama works through chunks to show progress.
- **Exit codes**: `0` when all processed files succeed, `1` whenever any error occurs (walk errors, I/O, LLM failure, etc.).

## 2. Processing logic / architecture
1. **Init**: parse flags, validate the required `rootPath`, derive the HTTP timeout (default 5m, overridable via `-request-timeout`), negotiate the model (override > autodetect > fallback list).
2. **Planning**: `filepath.WalkDir` captures every qualifying markdown file, derives the target filename, and applies exclusion regexes (`-exclude`), `-force`, `-dry-run`, and `-max-files` gates. Console chatter (status/progress lines) respects `-quiet`.
3. **Pipeline (`processFile`)**:
   - Read the entire markdown document.
   - Chunk text at rune boundaries respecting configured size/overlap.
   - For each chunk, build the chunk prompt, call `POST /api/generate`, and collect intermediate summaries.
   - Categorize the original length (SHORT/MEDIUM/LONG) based on rune count.
   - Feed chunk summaries plus the length hint into the final prompt to derive the cohesive markdown summary.
   - Honor `-dry-run` by skipping network calls and file writes while still reporting intended actions.
   - Write `<name>_summary.md` atomically, keeping the two-line ultra-short intro followed by the longer section layout.
4. **Error handling**: propagate recoverable errors per file, mark `hadError`, continue walking, and exit with code `1` if any errors were observed.
5. **Concurrency (optional)**: use a worker pool (default `min(max(NumCPU,1),4)`) fed by a channel of file plans, while synchronizing console output to maintain deterministic status lines.

## 3. Prompt templates

### 3.1 Chunk summary prompt
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

### 3.2 Final combined summary prompt
```
You are "Chief Summarizer", an assistant that creates structured summaries in the original language of the source text.

Task:
- You receive several partial summaries of different excerpts of ONE long markdown document.
- Combine them into ONE cohesive summary.
- Remove repetition and contradictions.
- Maintain the SAME LANGUAGE as the original text (usually German).
- Keep important names, dates and numbers.
- Be neutral and factual.

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

> Tip: prepend `Original document length category: SHORT|MEDIUM|LONG.` before the `Input:` block so the model can tune the layout.

## 4. Rough Go skeleton
- `cmd/chief-summarizer/main.go` wires together flag parsing, model selection, directory walking, and the future processing pipeline.
- `Config` holds every CLI option; `preferredModels` enumerates default model priorities.
- Helper stubs (`isMarkdown`, `isSummaryFile`, `summaryFilename`, `workersDefault`) isolate filesystem logic.
- `processFile` is intentionally left unimplemented; it will encapsulate reading, chunking, chunk summarization, final synthesis, and file writes.
- Future work: add `internal/chunking`, `internal/ollama`, and `internal/prompts` packages to keep responsibilities separated and simplify unit testing.
