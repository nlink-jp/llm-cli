# CLAUDE.md — llm-cli

**Organization rules (mandatory): https://github.com/nlink-jp/.github/blob/main/CONVENTIONS.md**

## This project

CLI client for local LLMs (LM Studio, Ollama) via OpenAI-compatible API endpoints.
Next-generation successor to lite-llm, fully integrated with nlk library.
Supports streaming, batch processing, multi-image VLM input, structured output (JSON schema),
and prompt injection protection.

## Key structure

```
main.go                  <- entry point, injects version
cmd/root.go              <- Cobra CLI flags + runPrompt orchestration
internal/config/         <- TOML config + LLM_CLI_* env var overrides
internal/client/         <- OpenAI-compatible API client (chat/completions)
internal/input/          <- stdin/file reading, image base64 encoding
internal/output/         <- text / json / jsonl formatting
```

## Build & test

```sh
make build    # dist/llm-cli
make test     # go test ./...
make check    # vet -> test -> build
```

## Design notes

- Uses nlk library (guard, jsonfix, strip, backoff, validate) — no custom implementations
- Data isolation: stdin/file input wrapped in `<user_data_{nonce}>` via nlk/guard
- response_format_strategy: auto (try native, fall back to prompt injection) / native / prompt
- Image input: `-i path` (repeatable), base64 data URI in OpenAI content array format
- Config priority: CLI flags -> env vars (LLM_CLI_*) -> config.toml -> defaults
- Module path: `github.com/nlink-jp/llm-cli`
