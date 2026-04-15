# RFP: llm-cli

> Generated: 2026-04-16
> Status: Draft

## 1. Problem Statement

A CLI tool for interacting with local LLMs (LM Studio / Ollama, OpenAI API-compatible endpoints).
Designed as a next-generation successor to lite-llm, fully leveraging the shared nlk library.

lite-llm will remain available, but future maintenance will be consolidated into llm-cli.
Key enhancements include multi-image input for VLM models, JSON schema-based structured output
(equivalent to gem-cli), and full integration with nlk (guard / jsonfix / strip / backoff / validate).

Built as a pipe-friendly UNIX CLI, designed for pipeline integration with other nlink-jp tools.

**Target user:** Internal use (pipeline integration with nlink-jp tool suite)

## 2. Functional Specification

### Commands / API Surface

```
llm-cli [flags] [prompt]

# Input
  -p, --prompt              Prompt text
  -f, --file                Input file or stdin (-)
  -s, --system-prompt       System prompt text
  -S, --system-prompt-file  System prompt file
  -i, --image               Image file path (repeatable, order preserved left-to-right)

# Model / Endpoint
  -m, --model               Model name
  --endpoint                API base URL

# Execution Modes
  --stream                  Streaming output
  --batch                   Line-by-line batch processing

# Output Formats
  --format                  text (default) | json | jsonl
  --json-schema             JSON Schema file path

# Security
  --no-safe-input           Disable data isolation (prompt injection defense)
  -q, --quiet               Suppress warnings
  --debug                   Debug output

# Configuration
  -c, --config              Config file path
```

**Image input:**
- Specify multiple images with `-i image1.png -i image2.jpg`, order preserved
- Sent as base64-encoded data URIs in OpenAI API-compatible format (content array + image_url)
- Supported formats: JPEG, PNG (GIF, WebP considered based on implementation complexity)
- Not compatible with batch mode (`--batch`)

### Input / Output

**Input:**
- Prompt: positional argument / `-p` flag / stdin / `-f` file
- System prompt: `-s` text / `-S` file
- Images: `-i` file path (repeatable)
- Batch mode: reads from stdin line-by-line

**Output:**
- `--format text`: Plain text (default)
- `--format json`: JSON output (repaired via nlk/jsonfix)
- `--format jsonl`: Batch mode only `{"input":"...","output":"...","error":null}`
- Streaming: `--stream` for token-by-token output (incompatible with `--json-schema`)

### Configuration

Config file: `~/.config/llm-cli/config.toml`

```toml
[api]
base_url = "http://localhost:1234/v1"   # LM Studio default
api_key = ""                             # For remote API connections
timeout_seconds = 120
response_format_strategy = "auto"        # auto | native | prompt

[model]
name = "default-model"
```

**Priority:** CLI flags > Environment variables (`LLM_CLI_*`) > config.toml > Defaults

### External Dependencies

- **LM Studio** / **Ollama**: OpenAI API-compatible endpoints (primary targets)
- **nlk**: guard, jsonfix, strip, backoff, validate (all 5 packages)
- **Cobra**: CLI framework
- Remote OpenAI API: connectable but not the primary use case

## 3. Design Decisions

**Language: Go**
- Standard language for nlink-jp CLI tools
- Direct integration with nlk library (Go)
- Same stack as lite-llm / gem-cli

**Full nlk integration:**
- Replaces lite-llm's custom isolation / JSON repair implementations with nlk packages
- `nlk/guard`: Prompt injection defense (128-bit nonce tags)
- `nlk/jsonfix`: LLM output JSON repair
- `nlk/strip`: Thinking/reasoning tag removal
- `nlk/backoff`: Exponential backoff retry for API errors
- `nlk/validate`: Rule-based validation for structured output

**response_format_strategy (carried over from lite-llm):**
- `auto`: Try native response_format first, fall back to prompt injection
- `native`: Always send response_format parameter
- `prompt`: Always inject format requirement into system prompt
- Ollama ignores json_schema format, making auto or prompt effectively required

**Out of scope:**
- Chat mode / session management (proved ineffective in lite-llm)
- Google Search Grounding (gem-cli specific)
- Context caching

## 4. Development Plan

### Phase 1: Core

- Project scaffolding (Makefile, go.mod, internal package structure)
- config.toml parsing + environment variable override + CLI flag integration
- Single prompt execution (blocking)
- Streaming output (SSE)
- `nlk/guard` data isolation
- `nlk/strip` + `nlk/jsonfix` output post-processing pipeline
- `nlk/backoff` API error retry
- Unit test suite

### Phase 2: Features

- `-i` multi-image input (base64, JPEG/PNG)
- `--json-schema` structured output + response_format_strategy
- `--batch` batch mode + JSONL output
- `nlk/validate` output validation
- Additional image format evaluation (GIF/WebP)

### Phase 3: Release

- README.md / README.ja.md
- CHANGELOG.md
- E2E testing (LM Studio + google/gemma-4-26b-a4b)
- AGENTS.md
- Release workflow (tag, gh release, cli-series submodule registration)

**Each phase can be reviewed independently.**

## 5. Required API Scopes / Permissions

- External OAuth scopes / IAM roles: **None**
- API key field included in config.toml (for remote API connections)
- Local LLMs (LM Studio / Ollama) require no authentication

## 6. Series Placement

**Series: cli-series**

Reason: Positioned as a CLI client for LLM service endpoints.
As a client tool for LM Studio / Ollama API endpoints, it aligns with
cli-series (scli, confl-cli, splunk-cli, gem-cli, etc.).

## 7. External Platform Constraints

### LM Studio

- **response_format**: `json_schema` supported (grammar-based sampling — GGUF: llama.cpp, MLX: Outlines)
- **json_object**: Supported (was unsupported in earlier versions)
- **Vision**: Supported (base64 data URI + image URL)
- **Default endpoint**: `http://localhost:1234/v1`
- **Responses API**: Experimental support (v0.3.29+)

### Ollama

- **response_format**: OpenAI-format `json_schema` is **ignored** (uses Ollama's own format parameter)
  - `response_format_strategy=prompt` fallback is effectively required
- **Vision**: base64 supported but not fully compatible with OpenAI format
  - Reports indicate data URI format within image_url in content array works
- **OpenAI-compatible API**: Experimental status, breaking changes possible
- **Default endpoint**: `http://localhost:11434/v1`

### Mitigation Strategy

- Carry over `response_format_strategy=auto` fallback mechanism from lite-llm
- Document prompt fallback as recommended setting for Ollama
- Unify image transmission using OpenAI format (content array + image_url + base64 data URI)
- Rely on Ollama's compatibility layer; no custom conversion layer

---

## Discussion Log

1. **Tool name & problem definition**: Confirmed direction as llm-cli, a next-gen CLI for local LLMs. lite-llm continues but maintenance shifts to llm-cli.

2. **Image support**: Confirmed multi-image support. Format: `-i path1 -i path2` with order preservation. Not compatible with batch mode.

3. **Chat mode**: Deemed unnecessary — proved ineffective in lite-llm.

4. **nlk integration**: All 5 packages (guard, jsonfix, strip, backoff, validate) utilized. backoff needed for LLM API errors. lite-llm's custom implementations consolidated into nlk.

5. **response_format_strategy**: Carried over auto/native/prompt strategy from lite-llm. Investigation confirmed Ollama ignores json_schema, validating this mechanism's importance.

6. **Series placement**: Decided on cli-series. Positioned as a CLI client for LLM service endpoints.

7. **E2E testing**: LM Studio + google/gemma-4-26b-a4b. Covered in Phase 3.

8. **Image formats**: JPEG/PNG required. GIF/WebP evaluated based on implementation complexity.

9. **External API constraint research**: LM Studio supports json_schema and Vision. Ollama has json_schema incompatibility and partial Vision support. Confirmed importance of fallback mechanism.
