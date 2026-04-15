# AGENTS.md — llm-cli

CLI client for local LLMs (LM Studio, Ollama) — streaming, batch, multi-image VLM, structured output.
Part of [cli-series](https://github.com/nlink-jp/cli-series).
Next-generation successor to [lite-llm](https://github.com/nlink-jp/lite-llm).

## Rules

- Project rules: -> [CLAUDE.md](CLAUDE.md)
- Organization conventions: -> [CONVENTIONS.md](https://github.com/nlink-jp/.github/blob/main/CONVENTIONS.md)

## Build & test

```sh
make build    # dist/llm-cli
make test     # go test ./...
make check    # vet -> test -> build
```

## Key structure

```
main.go                       <- entry point, injects version
cmd/root.go                   <- Cobra CLI flags, runSingle/runBatch orchestration
internal/config/config.go     <- TOML config + LLM_CLI_* env var overrides
internal/client/client.go     <- OpenAI-compatible HTTP client (blocking + SSE streaming)
internal/client/types.go      <- Request/response types (multimodal message support)
internal/input/input.go       <- stdin/file reading, source tracking
internal/input/image.go       <- Image loading, base64 encoding, MIME detection
internal/output/output.go     <- text / json / jsonl formatting
```

## Gotchas

- **nlk integration**: Uses nlk/guard, nlk/jsonfix, nlk/strip, nlk/backoff — no custom reimplementations.
- **Data isolation**: Enabled by default. Wraps stdin/file input in `<user_data_{nonce}>` via nlk/guard. Use `{{DATA_TAG}}` in system prompt. Disable with `--no-safe-input`.
- **response_format_strategy**: `auto` (default) tries native response_format, falls back to prompt injection. Ollama ignores json_schema — use `auto` or `prompt`.
- **Image input**: `-i path` repeatable, sent as base64 data URI in content array. Not compatible with `--batch`. Supported: JPEG, PNG.
- **Multimodal messages**: When images are present, user message Content is serialized as an array of content parts (text + image_url), not a plain string.
- **Stream vs non-stream**: Stream writes to stdout during generation. Non-stream returns full result then writes. Don't output twice.
- **Config priority**: CLI flags -> env vars (`LLM_CLI_*`) -> config file -> defaults.
- **Module path**: `github.com/nlink-jp/llm-cli`.
- **Env vars**: `LLM_CLI_BASE_URL`, `LLM_CLI_API_KEY`, `LLM_CLI_MODEL`, `LLM_CLI_RESPONSE_FORMAT_STRATEGY`.
