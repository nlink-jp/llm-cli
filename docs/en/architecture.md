# Architecture: llm-cli

> Last updated: 2026-04-16

## Why This Tool Exists

This tool succeeds [lite-llm](https://github.com/nlink-jp/lite-llm) (archived).
lite-llm was a capable OpenAI-compatible CLI, but its design predated the
[nlk](https://github.com/nlink-jp/nlk) shared library. As a result, lite-llm
carried its own implementations of prompt injection defense, JSON repair, and
thinking tag removal — duplicating functionality that nlk now provides as
tested, reusable packages.

llm-cli was built from scratch to:

- **Eliminate duplicate code** — replace custom isolation, JSON extraction, and
  tag stripping with nlk/guard, nlk/jsonfix, and nlk/strip
- **Add VLM support** — multi-image input (`-i`) for vision-language models,
  which lite-llm did not support
- **Add retry resilience** — nlk/backoff for transient API errors
- **Align with cli-series** — lite-llm was in lite-series; llm-cli belongs in
  cli-series as a service client for LLM API endpoints

## Why nlk Integration (not custom implementations)

lite-llm implemented its own:

| Concern | lite-llm implementation | llm-cli replacement |
|---------|------------------------|---------------------|
| Prompt injection defense | `internal/isolation/` (3-byte nonce) | nlk/guard (128-bit nonce) |
| JSON extraction & repair | `internal/output/` (ad-hoc parsing) | nlk/jsonfix (recursive descent parser) |
| Thinking tag removal | `internal/output/` (inline stripping) | nlk/strip (regex, multi-format) |
| Retry on errors | None | nlk/backoff (exponential + jitter) |
| Output validation | None | nlk/validate (rule-based) |

Benefits of consolidation:

- **Stronger security** — nlk/guard uses 128-bit nonces (vs. 24-bit in lite-llm),
  with ErrTagCollision defense-in-depth
- **Better JSON repair** — nlk/jsonfix handles single quotes, trailing commas,
  comments, unquoted keys, and escaped JSON that lite-llm's extraction missed
- **Single source of truth** — bug fixes in nlk benefit all consuming tools
  automatically

## Why Multimodal Messages (not separate image API)

OpenAI's chat/completions API supports multimodal input by embedding images as
base64 data URIs in the `content` array of user messages. This design was chosen
over alternatives:

| Alternative | Why rejected |
|-------------|-------------|
| Separate `/v1/images` endpoint | Not part of OpenAI chat API; doesn't exist in LM Studio/Ollama |
| Upload-then-reference | Requires state management; local LLMs don't support file uploads |
| URL-based image references | Requires the LLM server to fetch URLs; unreliable for local files |

Base64 inline is the only approach that works consistently across LM Studio,
Ollama, and remote OpenAI — no server-side fetching, no state, no extra endpoints.

**Image ordering matters** — images are appended to the content array in `-i`
flag order (left to right). Prompts like "Compare the first and second images"
rely on this ordering being deterministic.

## Why response_format_strategy

Local LLM servers differ in their `response_format` support:

| Server | `json_object` | `json_schema` | Behavior on unsupported |
|--------|:---:|:---:|---|
| LM Studio | Yes | Yes (grammar-based) | 400 error with descriptive message |
| Ollama | Partial | Ignored | Silently ignores the field |
| OpenAI | Yes | Yes | N/A (always supported) |

A single strategy cannot work for all backends. The three-mode approach
(inherited from lite-llm) handles this:

- **`auto`** — try native first; on 400/422 with keywords like "response_format"
  or "not supported", retry without it and inject the format requirement into
  the system prompt instead. This is the safe default.
- **`native`** — always send `response_format`. Use when you know the API
  supports it and want a hard failure if it doesn't.
- **`prompt`** — never send `response_format`. Always inject via system prompt.
  Recommended for Ollama, which silently ignores the field.

Fallback detection checks the error body for: `response_format`, `not supported`,
`unsupported`, `unknown field`.

## Why Go

- **Single-binary distribution** — critical for pipeline use
  (`echo "query" | llm-cli --format json | jq`)
- **nlk integration** — nlk is a Go library with zero external dependencies
- **cli-series consistency** — gem-cli, scli, splunk-cli are all Go
- **Cross-compilation** — `make build-all` produces 5 platform binaries
  with `CGO_ENABLED=0`

## Security

| Concern | Mitigation |
|---------|-----------|
| Prompt injection | stdin/file input wrapped with nlk/guard nonce-tagged XML (128-bit entropy) |
| Tag collision | ErrTagCollision check as defense-in-depth (probability: 2^-128) |
| Credential exposure | Config file permission check warns on `perm & 0077 != 0` |
| API key in transit | Bearer token via Authorization header (HTTPS recommended) |
| Image data leakage | Images sent inline as base64; no intermediate storage or caching |
| LLM output manipulation | nlk/strip removes thinking tags before JSON extraction |

## Data Flow

```
Input Sources                    Processing Pipeline                Output
─────────────                    ───────────────────                ──────
-p flag  ──┐                     ┌──────────────────┐
positional ─┤ ReadUserInput ──→ │ nlk/guard.Wrap    │──→ API Request
-f file  ──┤ (source tracking)  │ (if SourceExternal)│    ┌──────────┐
stdin    ──┘                     └──────────────────┘    │ OpenAI   │
                                                         │ compat   │
-s flag  ──┐                     ┌──────────────────┐    │ endpoint │
-S file  ──┤ ReadSystemPrompt → │ guard.Expand      │──→ │          │
            └                    │ ({{DATA_TAG}})    │    └────┬─────┘
                                 └──────────────────┘         │
-i paths ──→ LoadImages ──→ base64 encode ──→ content array   │
                                                               ▼
                                 ┌──────────────────┐    Response
                                 │ nlk/strip        │←──────┘
                                 │ nlk/jsonfix      │
                                 │ output.Formatter │──→ stdout
                                 └──────────────────┘
```
