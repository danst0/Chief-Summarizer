# Copilot Instructions for Chief Summarizer

## Project Overview
- **Type**: Go CLI tool
- **Purpose**: Batch-generate structured markdown summaries using local Ollama models
- **Status**: Production-ready

## Completed Setup Steps
- [x] Project structure created
- [x] Core CLI implementation (flag parsing, file discovery, chunking, summarization)
- [x] Ollama integration with automatic model selection
- [x] Makefile for build/install
- [x] Systemd automation units
- [x] Documentation (README.md)
- [x] Config file validation

## Development Guidelines

### Workflow
- Work through tasks systematically
- Keep communication concise and focused
- Follow Go best practices and idiomatic patterns

### Project Structure
- Use current directory as project root
- Main code in `cmd/chief-summarizer/main.go`
- Keep systemd units under `systemd/`

### Code Standards
- Follow standard Go formatting (gofmt)
- Handle errors explicitly
- Use meaningful variable names
- Add comments for complex logic

### Testing & Validation
- Build with `make build` after changes
- Test with `-dry-run` flag before production use
- Verify systemd units with `systemctl --user status`

## Current State
- Project is successfully compiled and operational
- README.md is up to date with comprehensive documentation
- Systemd automation configured for scheduled runs
- Config file support implemented (optional YAML configuration)
