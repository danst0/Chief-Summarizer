package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

var preferredModels = []string{"qwen2.5:14b", "llama3.1:8b", "mistral:7b"}
var errMaxFiles = errors.New("max-files limit reached")

// Config captures all runtime options parsed from CLI flags.
type Config struct {
	RootDir      string
	Host         string
	Model        string
	ChunkSize    int
	ChunkOverlap int
	Force        bool
	DryRun       bool
	MaxFiles     int
	Verbose      bool
}

func main() {
	cfg := parseFlags()

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
			fmt.Printf("ERR  %s (walk error: %v)\n", path, walkErr)
			hadError = true
			return nil
		}
		if d.IsDir() || !isMarkdown(path) || isSummaryFile(path) {
			return nil
		}

		summaryPath := summaryFilename(path)
		if !cfg.Force {
			if _, err := os.Stat(summaryPath); err == nil {
				fmt.Printf("SKIP %s (summary exists)\n", path)
				return nil
			}
		}

		if cfg.MaxFiles > 0 && processed >= cfg.MaxFiles {
			return errMaxFiles
		}

		if cfg.DryRun {
			fmt.Printf(
				"DRY  %s (would create %s, model=%s, chunk=%d/%d)\n",
				path, summaryPath, cfg.Model, cfg.ChunkSize, cfg.ChunkOverlap,
			)
			processed++
			return nil
		}

		if err := processFile(path, summaryPath, cfg); err != nil {
			fmt.Printf("ERR  %s (%v)\n", path, err)
			hadError = true
		} else {
			fmt.Printf("OK   %s -> %s\n", path, summaryPath)
		}
		processed++
		return nil
	})

	if err != nil && !errors.Is(err, errMaxFiles) {
		fmt.Fprintf(os.Stderr, "ERR  walk error: %v\n", err)
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
	version := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *version {
		fmt.Println("chief-summarizer v0.1.0")
		os.Exit(0)
	}

	if flag.NArg() > 0 {
		cfg.RootDir = flag.Arg(0)
	} else {
		cfg.RootDir = "."
	}

	return cfg
}

func chooseModel(cfg Config) (string, error) {
	if cfg.Model != "" {
		return cfg.Model, nil
	}
	// TODO: call Ollama /api/tags and pick the best available model.
	if len(preferredModels) == 0 {
		return "", errors.New("no preferred models configured")
	}
	return preferredModels[0], nil
}

func processFile(path, summaryPath string, cfg Config) error {
	// TODO: implement full pipeline (read, chunk, summarize, merge, write)
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
