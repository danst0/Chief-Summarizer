package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var preferredModels = []string{"qwen3:14b", "deepseek-r1:14b", "llama3"}
var errMaxFiles = errors.New("max-files limit reached")

var httpClient = &http.Client{Timeout: 360 * time.Second}

// Config captures all runtime options parsed from CLI flags.
type Config struct {
	RootDir        string
	Host           string
	Model          string
	ChunkSize      int
	ChunkOverlap   int
	Force          bool
	DryRun         bool
	MaxFiles       int
	Verbose        bool
	Quiet          bool
	Excludes       []*regexp.Regexp
	RequestTimeout time.Duration
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	cfg := parseFlags()
	httpClient.Timeout = cfg.RequestTimeout

	model, err := chooseModel(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERR  model selection failed: %v\n", err)
		os.Exit(1)
	}
	cfg.Model = model

	hadError := false
	processed := 0

	err = filepath.WalkDir(cfg.RootDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			errorf("ERR  %s (walk error: %v)\n", path, walkErr)
			hadError = true
			return nil
		}
		display := displayPath(path, cfg.RootDir)
		if matchesExclude(path, cfg.RootDir, cfg.Excludes) {
			if d.IsDir() {
				if cfg.Verbose {
					statusf(cfg, "SKIP %s (directory excluded)\n", display)
				}
				return filepath.SkipDir
			}
			statusf(cfg, "SKIP %s (excluded by pattern)\n", display)
			return nil
		}
		if d.IsDir() || !isMarkdown(path) || isSummaryFile(path) {
			return nil
		}

		summaryPath := summaryFilename(path)
		if !cfg.Force {
			if _, err := os.Stat(summaryPath); err == nil {
				statusf(cfg, "SKIP %s (summary exists)\n", display)
				return nil
			}
		}

		if cfg.MaxFiles > 0 && processed >= cfg.MaxFiles {
			return errMaxFiles
		}

		if cfg.DryRun {
			statusf(
				cfg,
				"DRY  %s (would create %s, model=%s, chunk=%d/%d)\n",
				display, displayPath(summaryPath, cfg.RootDir), cfg.Model, cfg.ChunkSize, cfg.ChunkOverlap,
			)
			processed++
			return nil
		}

		if err := processFile(path, summaryPath, cfg); err != nil {
			errorf("ERR  %s (%v)\n", display, err)
			hadError = true
		} else {
			statusf(cfg, "OK   %s -> %s\n", display, displayPath(summaryPath, cfg.RootDir))
		}
		processed++
		return nil
	})

	if err != nil && !errors.Is(err, errMaxFiles) {
		errorf("ERR  walk error: %v\n", err)
		hadError = true
	}

	if hadError {
		os.Exit(1)
	}
}

func parseFlags() Config {
	var cfg Config
	flag.StringVar(&cfg.Host, "host", "http://localhost:11434", "Ollama host URL")
	flag.StringVar(&cfg.Model, "model", "", "Model name (optional)")
	flag.IntVar(&cfg.ChunkSize, "chunk-size", 4000, "Chunk size in characters")
	flag.IntVar(&cfg.ChunkOverlap, "chunk-overlap", 400, "Chunk overlap in characters")
	flag.BoolVar(&cfg.Force, "force", false, "Overwrite existing *_summary.md files")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "Dry run (no LLM calls, no writes)")
	flag.IntVar(&cfg.MaxFiles, "max-files", 0, "Max files to process (0 = unlimited)")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&cfg.Quiet, "quiet", false, "Suppress progress/status output (errors still reported)")
	flag.DurationVar(&cfg.RequestTimeout, "request-timeout", 5*time.Minute, "HTTP request timeout (e.g. 300s, 5m)")
	var excludePatterns multiFlag
	flag.Var(&excludePatterns, "exclude", "Regular expression for paths to skip (repeatable)")
	version := flag.Bool("version", false, "Print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: chief-summarizer [flags] <root-path>\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *version {
		fmt.Println("chief-summarizer v0.1.0")
		os.Exit(0)
	}

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "ERR  root path argument is required")
		flag.Usage()
		os.Exit(2)
	}
	cfg.RootDir = flag.Arg(0)
	if _, err := os.Stat(cfg.RootDir); err != nil {
		fmt.Fprintf(os.Stderr, "ERR  invalid root path %q: %v\n", cfg.RootDir, err)
		os.Exit(1)
	}
	if len(excludePatterns) > 0 {
		cfg.Excludes = make([]*regexp.Regexp, 0, len(excludePatterns))
		for _, pattern := range excludePatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERR  invalid -exclude pattern %q: %v\n", pattern, err)
				os.Exit(2)
			}
			cfg.Excludes = append(cfg.Excludes, re)
		}
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 5 * time.Minute
	}

	return cfg
}

func chooseModel(cfg Config) (string, error) {
	if cfg.Model != "" {
		return cfg.Model, nil
	}
	available, err := listAvailableModels(cfg.Host)
	if err != nil {
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "WARN unable to query models from %s: %v\n", cfg.Host, err)
		}
		if len(preferredModels) == 0 {
			return "", errors.New("no preferred models configured")
		}
		return preferredModels[0], nil
	}
	if len(available) == 0 {
		if len(preferredModels) == 0 {
			return "", errors.New("no preferred models configured")
		}
		return preferredModels[0], nil
	}
	availableSet := make(map[string]struct{}, len(available))
	for _, name := range available {
		availableSet[name] = struct{}{}
	}
	for _, preferred := range preferredModels {
		if _, ok := availableSet[preferred]; ok {
			return preferred, nil
		}
		if match, ok := findClosestModel(preferred, available); ok {
			if cfg.Verbose {
				fmt.Fprintf(os.Stderr, "INFO using closest installed model %s for preferred %s\n", match, preferred)
			}
			return match, nil
		}
	}
	fallback := available[0]
	fmt.Fprintf(os.Stderr, "WARN none of the preferred models %v are installed; using %s instead\n", preferredModels, fallback)
	return fallback, nil
}

func processFile(path, summaryPath string, cfg Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	content := string(data)
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return fmt.Errorf("file is empty")
	}
	chunks := chunkText(trimmed, cfg.ChunkSize, cfg.ChunkOverlap)
	if len(chunks) == 0 {
		chunks = []string{trimmed}
	}
	chunkSummaries := make([]string, 0, len(chunks))
	for idx, chunk := range chunks {
		statusf(cfg, "CHNK %s (%d/%d)\n", displayPath(path, cfg.RootDir), idx+1, len(chunks))
		prompt := buildChunkPrompt(chunk)
		resp, err := callOllama(cfg.Host, cfg.Model, prompt)
		if err != nil {
			return fmt.Errorf("chunk %d summarization failed: %w", idx+1, err)
		}
		chunkSummaries = append(chunkSummaries, strings.TrimSpace(resp))
	}
	lengthCategory := lengthCategoryFromRunes(len([]rune(trimmed)))
	finalPrompt := buildFinalPrompt(chunkSummaries, lengthCategory)
	statusf(cfg, "MERGE %s (%d chunks)\n", displayPath(path, cfg.RootDir), len(chunkSummaries))
	finalSummary, err := callOllama(cfg.Host, cfg.Model, finalPrompt)
	if err != nil {
		return fmt.Errorf("final summary failed: %w", err)
	}
	if err := os.WriteFile(summaryPath, []byte(strings.TrimSpace(finalSummary)+"\n"), 0o644); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	return nil
}

func isMarkdown(path string) bool {
	return filepath.Ext(path) == ".md"
}

func isSummaryFile(path string) bool {
	base := filepath.Base(path)
	return len(base) > len("_summary.md") && base[len(base)-len("_summary.md"):] == "_summary.md"
}

func summaryFilename(path string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	return filepath.Join(dir, name+"_summary"+ext)
}

func statusf(cfg Config, format string, args ...any) {
	if cfg.Quiet {
		return
	}
	fmt.Printf(format, args...)
}

func errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func workersDefault() int {
	n := runtime.NumCPU()
	if n < 2 {
		return 1
	}
	if n > 4 {
		return 4
	}
	return n
}

func chunkText(text string, size, overlap int) []string {
	runes := []rune(text)
	if size <= 0 {
		size = 1000
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 4
	}
	chunks := []string{}
	for start := 0; start < len(runes); {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) {
			break
		}
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}
	return chunks
}

func buildChunkPrompt(chunk string) string {
	var b strings.Builder
	b.WriteString("You are \"Chief Summarizer\", an assistant that creates concise summaries in the original language of the text.\n\n")
	b.WriteString("Task:\n- Read the following markdown excerpt.\n- Write a short summary of this excerpt.\n- Use the SAME LANGUAGE as the text (usually German).\n- Keep names, dates and key facts accurate.\n- Do NOT add your own interpretations or new ideas.\n- Do NOT write an overall document summary, only summarize THIS excerpt.\n\n")
	b.WriteString("Output format:\n- 1 short paragraph in plain text (no headings).\n- Maximum ~120 words.\n\nExcerpt:\n---\n")
	b.WriteString(chunk)
	b.WriteString("\n---\n")
	return b.String()
}

func buildFinalPrompt(chunkSummaries []string, lengthCategory string) string {
	var b strings.Builder
	b.WriteString("You are \"Chief Summarizer\", an assistant that creates structured summaries in the original language of the source text.\n\n")
	b.WriteString("Task:\n- You receive several partial summaries of different excerpts of ONE long markdown document.\n- Combine them into ONE cohesive summary.\n- Remove repetition and contradictions.\n- Maintain the SAME LANGUAGE as the original text (usually German).\n- Keep important names, dates and numbers.\n- Be neutral and factual.\n\n")
	b.WriteString("Output format (Markdown, fixed):\n\n1. First, write a very short two-line \"Ultra-Kurzfassung\" overview:\n   - Line 1: one short sentence describing the main topic.\n   - Line 2: one short sentence describing the main outcome or conclusion.\n\n2. Then add a blank line.\n\n3. Then write the \"Ausführliche Zusammenfassung\" (detailed summary) in markdown:\n   - If the original document was short (~< 1.500 Wörter):\n     - write 2–4 short paragraphs OR 3–6 bullet points.\n   - If the original document was medium (1.500–5.000 Wörter):\n     - write 3–6 paragraphs and optionally 3–8 bullet points.\n   - If the original document was long (> 5.000 Wörter):\n     - use clear markdown headings (##) and bullet lists for structure.\n   - Always stay focused on the key points, decisions, arguments, and results.\n\n")
	b.WriteString(fmt.Sprintf("Original document length category: %s.\n\n", lengthCategory))
	b.WriteString("Input:\nThe following are partial summaries of the document, in order:\n\n---\n")
	for i, summary := range chunkSummaries {
		b.WriteString(fmt.Sprintf("Chunk %d:\n%s\n\n", i+1, summary))
	}
	b.WriteString("---\n\nNow produce ONLY the markdown summary as specified above.\nDo not add any intro text or explanations around it.\n")
	return b.String()
}

func lengthCategoryFromRunes(count int) string {
	switch {
	case count < 8000:
		return "SHORT"
	case count < 25000:
		return "MEDIUM"
	default:
		return "LONG"
	}
}

func listAvailableModels(host string) ([]string, error) {
	endpoint := strings.TrimRight(host, "/") + "/api/tags"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("ollama tags request failed: %s: %s", resp.Status, bytes.TrimSpace(body))
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	available := make([]string, 0, len(payload.Models))
	for _, m := range payload.Models {
		available = append(available, m.Name)
	}
	return available, nil
}

func callOllama(host, model, prompt string) (string, error) {
	endpoint := strings.TrimRight(host, "/") + "/api/generate"
	body, err := json.Marshal(map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return "", fmt.Errorf("ollama generate failed: %s: %s", resp.Status, bytes.TrimSpace(payload))
	}
	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if strings.TrimSpace(result.Response) == "" {
		return "", errors.New("ollama returned empty response")
	}
	return result.Response, nil
}

func findClosestModel(preferred string, available []string) (string, bool) {
	base := baseModelName(preferred)
	best := ""
	bestScore := 0
	for _, candidate := range available {
		if candidate == preferred {
			return candidate, true
		}
		score := modelSimilarityScore(base, baseModelName(candidate))
		if score > bestScore {
			best = candidate
			bestScore = score
		}
	}
	if bestScore > 0 {
		return best, true
	}
	return "", false
}

func baseModelName(name string) string {
	if idx := strings.IndexRune(name, ':'); idx >= 0 {
		return name[:idx]
	}
	return name
}

func modelSimilarityScore(basePreferred, baseCandidate string) int {
	if basePreferred == baseCandidate {
		return 3
	}
	if strings.HasPrefix(baseCandidate, basePreferred) || strings.HasPrefix(basePreferred, baseCandidate) {
		return 2
	}
	if strings.Contains(baseCandidate, basePreferred) || strings.Contains(basePreferred, baseCandidate) {
		return 1
	}
	return 0
}

func matchesExclude(path, root string, patterns []*regexp.Regexp) bool {
	if len(patterns) == 0 {
		return false
	}
	candidates := []string{path}
	if rel, err := filepath.Rel(root, path); err == nil {
		if rel == "." {
			rel = filepath.Base(path)
		}
		candidates = append(candidates, rel)
	}
	for _, candidate := range candidates {
		for _, re := range patterns {
			if re.MatchString(candidate) {
				return true
			}
		}
	}
	return false
}

func displayPath(path, root string) string {
	if rel, err := filepath.Rel(root, path); err == nil && rel != "." {
		return rel
	}
	return path
}
