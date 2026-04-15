# 構造化出力ガイド

## 概要

llm-cli は3つの構造化出力モードをサポートする:

| モード | フラグ | API メカニズム | 信頼性 |
|--------|--------|---------------|--------|
| JSON オブジェクト | `--format json` | `response_format: {type: json_object}` | 高（ネイティブ） |
| JSON スキーマ | `--json-schema <file>` | `response_format: {type: json_schema, ...}` | 最高（ネイティブ） |
| プロンプト注入 | `response_format_strategy = prompt` | システムプロンプトへの指示のみ | 低 |

## `--format json` — JSON オブジェクト出力

モデルに有効な JSON オブジェクトを要求する。API がフォーマットをネイティブに強制する。

```sh
llm-cli --format json "5つのプログラミング言語を作成年付きで出力してください"
```

**重要な制約（OpenAI API 要件）:** `json_object` モード使用時、プロンプトの
どこか（システムプロンプトまたはユーザープロンプト）に "JSON" という単語を
含める*必要がある*。含めないと OpenAI API がエラーを返す場合がある。
llm-cli は自動で "JSON" を追加しない。

## `--json-schema <file>` — スキーマ制約付き出力

JSON スキーマに厳密に準拠した出力を要求する。予測可能な形状の構造化データを
取得する最も信頼性の高い方法。

```sh
llm-cli --json-schema person.json "架空の人物を生成してください"
```

### スキーマファイルの書き方

スキーマファイルは JSON スキーマオブジェクトを含む有効な JSON ドキュメントでなければならない。

例 `person.json`:

```json
{
  "type": "object",
  "properties": {
    "name": { "type": "string" },
    "age": { "type": "integer", "minimum": 0 },
    "email": { "type": "string", "format": "email" }
  },
  "required": ["name", "age"],
  "additionalProperties": false
}
```

### OpenAI strict モードの制約

`--json-schema` 使用時、llm-cli は常に `strict: true` を送信する。
OpenAI API は strict モードで以下の制約を課す:

- すべてのオブジェクトプロパティを `required` に列挙すること
- `additionalProperties` は `false` であること
- 再帰スキーマは `$defs` を使用すること
- `anyOf`、`oneOf`、`allOf` は制限付きでサポート
- 一部の JSON Schema キーワードは非対応（例: `if/then/else`、`not`）

詳細は [OpenAI 構造化出力ドキュメント](https://platform.openai.com/docs/guides/structured-outputs)
を参照。

### スキーマ名

API に送信されるスキーマ名は `user_schema`。ファイル名は自身の整理のために
わかりやすい名前をつけること。

### システムプロンプトとの併用

`--json-schema` 使用時でも、期待する出力を平文で記述するとシステムプロンプトで
品質が向上する:

```sh
llm-cli \
  --json-schema person.json \
  -s "ファンタジー小説にふさわしい架空の人物を生成してください。" \
  "キャラクターを作成"
```

### `--json-schema` とバッチモード

`--json-schema` と `--batch` は併用できない。`--json-schema` は JSON フォーマットを
暗示するが、`--batch` は `--format jsonl` を必要とするため。代わりに `--format jsonl`
と `--batch` を使用し、スキーマの指示はシステムプロンプトに含めること。

## フォールバック戦略 (`response_format_strategy`)

ローカル LLM API（LM Studio、Ollama 等）は `response_format` フィールドに
非対応で、このフィールドが存在すると 4xx エラーを返す場合がある。
`~/.config/llm-cli/config.toml` でフォールバック動作を設定する:

### `auto`（デフォルト）

`response_format` を送信。API が拒否した場合、プロンプト注入でリトライし
stderr に警告を出力。

```toml
response_format_strategy = "auto"
```

フォールバック検出はエラーボディで以下のパターンを確認:
`response_format`、`not supported`、`unsupported`、`unknown field`。

### `native`

常に `response_format` を送信。API が非対応の場合はコマンド失敗。
構造化出力対応が確認済みの OpenAI 等の API に使用。

```toml
response_format_strategy = "native"
```

### `prompt`

`response_format` を送信しない。常にシステムプロンプトへの注入のみ使用。
`response_format` に非対応のローカル LLM（Ollama）に推奨。

```toml
response_format_strategy = "prompt"
```

**prompt 戦略の制限事項:**

- JSON 出力は保証されない。モデルが JSON の前後に散文を含む可能性がある。
- スキーマ準拠はベストエフォート。複雑なスキーマは正確に従わない場合がある。
- `--json-schema` は引き続き動作するが、スキーマ強制はモデルの指示追従能力に
  完全に依存する。

**自動 JSON 抽出:**

モデルが JSON ペイロードの周囲に余分なコンテンツを出力した場合、llm-cli は
nlk/jsonfix を使用して自動的に除去・抽出する。対応パターン:

| パターン | 対象モデル例 |
|----------|-------------|
| `<think>...</think>` / `<reasoning>...</reasoning>` | DeepSeek R1、Qwen3、QwQ |
| `[THINK]...[/THINK]` | Mistral (Magistral、Ministral-3、Devstral) |
| Markdown コードフェンス（` ```json ``` `、` ``` ``` `） | 各種 |
| シングルクォートキー、末尾カンマ、コメント | 各種 |
| 制御トークンや `{` / `[` 前のプレーンテキスト | 各種 |

有効な JSON が見つからない場合、レスポンス全体が JSON 文字列として出力される。

## `--format jsonl` — JSON Lines（バッチ専用）

`--format jsonl` は `--batch` が**必須**。`--batch` なしで `--format jsonl` を
指定するとエラー。処理された各行は1つの JSON Lines レコードを生成:

```jsonl
{"input":"入力テキスト","output":"モデル応答","error":null}
```

エラー時:

```jsonl
{"input":"入力テキスト","output":null,"error":"API error (status 429): rate limit exceeded"}
```

このフォーマットは `jq` 等のツールによる下流処理を想定:

```sh
cat results.jsonl | jq -r '.output' | head -5
```

## 非互換の組み合わせ

| フラグの組み合わせ | 結果 | 理由 |
|-------------------|------|------|
| `--json-schema` + `--stream` | エラー | スキーマ検証には完全なレスポンスが必要 |
| `--format jsonl` + `--batch` なし | エラー | JSONL はバッチ出力フォーマット |
| `--image` + `--batch` | エラー | バッチモードはテキスト行のみ処理 |
