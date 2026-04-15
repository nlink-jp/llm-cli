# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.1.0] - 2026-04-16

### Added

- Single prompt execution (blocking and streaming)
- Line-by-line batch processing with JSONL output (`--batch`, `--format jsonl`)
- Multi-image input for VLM models (`-i`, repeatable, JPEG/PNG)
- JSON schema structured output (`--json-schema`)
- JSON output with automatic repair (`--format json`)
- `response_format_strategy` with auto/native/prompt modes
  - `auto`: try native `response_format`, fall back to prompt injection
  - `native`: always send `response_format`
  - `prompt`: always use system prompt injection
- Prompt injection protection via nlk/guard (enabled by default)
- Exponential backoff retry via nlk/backoff
- LLM output post-processing: nlk/strip (thinking tags) + nlk/jsonfix (JSON repair)
- TOML config (`~/.config/llm-cli/config.toml`) with env var overrides (`LLM_CLI_*`)
- Config file permission check (warns on insecure permissions)
- Debug mode (`--debug`) for request/response inspection
- Signal handling (graceful Ctrl+C cancellation)
