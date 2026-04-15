# llm-cli

CLI client for local LLMs (LM Studio, Ollama) via OpenAI-compatible API endpoints.

Next-generation successor to [lite-llm](https://github.com/nlink-jp/lite-llm),
fully integrated with the [nlk](https://github.com/nlink-jp/nlk) library.

## Features

- Single prompt and streaming output
- Line-by-line batch processing with JSONL output
- Multi-image input for VLM models (`-i`)
- JSON schema structured output (`--json-schema`)
- Automatic response_format fallback for local LLMs
- Prompt injection protection via nlk/guard (enabled by default)
- Exponential backoff retry via nlk/backoff
- Flexible config: config.toml / environment variables / CLI flags

## Installation

```bash
make build
cp dist/llm-cli /usr/local/bin/
```

## Configuration

Copy the example config and adjust:

```bash
mkdir -p ~/.config/llm-cli
cp config.example.toml ~/.config/llm-cli/config.toml
chmod 600 ~/.config/llm-cli/config.toml
```

### Config file (`~/.config/llm-cli/config.toml`)

```toml
[api]
base_url = "http://localhost:1234/v1"   # LM Studio default
api_key = ""                             # For remote APIs
timeout_seconds = 120
response_format_strategy = "auto"        # auto | native | prompt

[model]
name = "default-model"
```

### Response format strategy

| Strategy | Behavior |
|----------|----------|
| `auto` (default) | Send `response_format` to API; fall back to prompt injection if unsupported |
| `native` | Always send `response_format`; fail if API rejects it |
| `prompt` | Never send `response_format`; inject format requirement into system prompt |

Ollama ignores OpenAI-format `json_schema` — use `auto` or `prompt`.

### Environment variables

| Variable | Overrides |
|----------|-----------|
| `LLM_CLI_BASE_URL` | `api.base_url` |
| `LLM_CLI_API_KEY` | `api.api_key` |
| `LLM_CLI_MODEL` | `model.name` |
| `LLM_CLI_RESPONSE_FORMAT_STRATEGY` | `api.response_format_strategy` |

**Priority:** CLI flags > environment variables > config file > defaults

## Usage

```bash
# Basic prompt
llm-cli "Explain what a goroutine is"

# With system prompt and streaming
llm-cli -s "You are a Go expert" -p "Explain channels" --stream

# Pipe input with data isolation
echo "Review this code" | llm-cli -s "You are a code reviewer"

# Multi-image input (VLM)
llm-cli -i photo1.jpg -i photo2.png "Compare these two images"

# Structured JSON output with schema
llm-cli --json-schema schema.json "Extract entities from: ..."

# JSON output without schema
llm-cli --format json "List 3 colors as JSON array"

# Batch processing
cat prompts.txt | llm-cli --batch --format jsonl -s "Translate to Japanese"

# Debug mode (show request/response)
llm-cli --debug "Say hi"

# Use specific model and endpoint
llm-cli -m "google/gemma-4-26b-a4b" --endpoint "http://localhost:11434" "Hello"
```

### Flags

```
Input:
  -p, --prompt              Prompt text
  -f, --file                Input file path (use - for stdin)
  -s, --system-prompt       System prompt text
  -S, --system-prompt-file  System prompt file path
  -i, --image               Image file path (repeatable, order preserved)

Model / Endpoint:
  -m, --model               Model name
      --endpoint            API base URL

Execution Modes:
      --stream              Enable streaming output
      --batch               Enable line-by-line batch processing

Output Formats:
      --format              text (default) | json | jsonl
      --json-schema         JSON Schema file path for structured output

Security:
      --no-safe-input       Disable prompt injection protection
  -q, --quiet               Suppress warnings
      --debug               Enable debug output

Configuration:
  -c, --config              Config file path
```

### Constraints

- `--stream` and `--batch` are mutually exclusive
- `--format jsonl` requires `--batch`
- `--json-schema` and `--stream` are incompatible
- `--image` and `--batch` are incompatible
- Supported image formats: JPEG (`.jpg`, `.jpeg`), PNG (`.png`)

### Data isolation

By default, piped stdin and file input (`-f`) are wrapped in nonce-tagged XML
to prevent prompt injection. Use `{{DATA_TAG}}` in the system prompt to
reference the tag name. Disable with `--no-safe-input`.

## Building

```bash
make build      # dist/llm-cli (current platform)
make build-all  # All platforms (linux/darwin/windows, amd64/arm64)
make test       # Run tests
make check      # vet + test + build
```

## Documentation

- [Architecture](docs/en/architecture.md) — design rationale and data flow
- [Structured Output Guide](docs/en/structured-output.md) — JSON schema, fallback strategies
- [RFP](docs/en/llm-cli-rfp.md) — requirements and planning document

## License

MIT License. See [LICENSE](LICENSE) for details.
