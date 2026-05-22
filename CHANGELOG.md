# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.1.2] - 2026-05-22

### Added

- **Pre-built binary releases for the first time.** A new `package`
  target produces zipped binaries for darwin/amd64, darwin/arm64,
  linux/amd64, linux/arm64, and windows/amd64. Previously llm-cli
  was installed via `go install` only. Asset naming:
  `llm-cli-vX.Y.Z-<os>-<arch>.zip`.
- **Darwin builds are Developer ID signed and Apple-notarized.**
  `make package` runs `scripts/codesign-darwin.sh` per darwin
  binary and `scripts/notarize-darwin.sh` per darwin zip,
  following the org-wide convention in `nlink-jp/.github`
  CONVENTIONS.md §Code Signing. End users on macOS no longer
  need to bypass Gatekeeper with right-click → Open; local
  Dropbox-synced (FileProvider-managed) install paths no longer
  SIGKILL the binary on launch.

No behaviour change to the binary itself — feature-wise this is
identical to v0.1.1.

## [0.1.1] - 2026-05-03

### Fixed

- Bump nlk to v0.5.2 to pick up the strip fix: think-tag handling
  no longer truncates LLM responses that explain the literal
  `<think>` tag inside a markdown inline-code span.

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
