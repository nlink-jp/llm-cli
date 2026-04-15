# llm-cli

ローカルLLM（LM Studio, Ollama）をOpenAI互換APIで利用するCLIクライアント。

[lite-llm](https://github.com/nlink-jp/lite-llm) の新世代後継。
[nlk](https://github.com/nlink-jp/nlk) ライブラリを全面統合。

## 機能

- 単発プロンプト実行とストリーミング出力
- 行単位バッチ処理（JSONL出力）
- VLMモデル向けマルチイメージ入力（`-i`）
- JSONスキーマによる構造化出力（`--json-schema`）
- ローカルLLM向け response_format 自動フォールバック
- nlk/guard によるプロンプトインジェクション防御（デフォルト有効）
- nlk/backoff による指数バックオフリトライ
- 柔軟な設定: config.toml / 環境変数 / CLIフラグ

## インストール

```bash
make build
cp dist/llm-cli /usr/local/bin/
```

## 設定

設定ファイルのテンプレートをコピーして編集:

```bash
mkdir -p ~/.config/llm-cli
cp config.example.toml ~/.config/llm-cli/config.toml
chmod 600 ~/.config/llm-cli/config.toml
```

### 設定ファイル (`~/.config/llm-cli/config.toml`)

```toml
[api]
base_url = "http://localhost:1234/v1"   # LM Studio デフォルト
api_key = ""                             # リモートAPI用
timeout_seconds = 120
response_format_strategy = "auto"        # auto | native | prompt

[model]
name = "default-model"
```

### Response format strategy

| 戦略 | 動作 |
|------|------|
| `auto`（デフォルト） | `response_format` をAPIに送信。非対応ならプロンプト注入にフォールバック |
| `native` | 常に `response_format` を送信。API非対応ならエラー |
| `prompt` | `response_format` を送信しない。常にシステムプロンプトへの注入で対応 |

Ollama は OpenAI形式の `json_schema` を無視するため、`auto` または `prompt` を使用。

### 環境変数

| 変数 | 上書き対象 |
|------|-----------|
| `LLM_CLI_BASE_URL` | `api.base_url` |
| `LLM_CLI_API_KEY` | `api.api_key` |
| `LLM_CLI_MODEL` | `model.name` |
| `LLM_CLI_RESPONSE_FORMAT_STRATEGY` | `api.response_format_strategy` |

**優先順位:** CLIフラグ > 環境変数 > 設定ファイル > デフォルト値

## 使い方

```bash
# 基本的なプロンプト
llm-cli "goroutineとは何か説明してください"

# システムプロンプト付き + ストリーミング
llm-cli -s "あなたはGoの専門家です" -p "チャネルについて説明してください" --stream

# パイプ入力（データ隔離あり）
echo "このコードをレビューしてください" | llm-cli -s "あなたはコードレビュアーです"

# マルチイメージ入力（VLM）
llm-cli -i photo1.jpg -i photo2.png "この2枚の画像を比較してください"

# JSONスキーマによる構造化出力
llm-cli --json-schema schema.json "エンティティを抽出: ..."

# スキーマなしのJSON出力
llm-cli --format json "3色をJSON配列で出力してください"

# バッチ処理
cat prompts.txt | llm-cli --batch --format jsonl -s "日本語に翻訳してください"

# デバッグモード（リクエスト/レスポンス表示）
llm-cli --debug "こんにちは"

# モデル・エンドポイント指定
llm-cli -m "google/gemma-4-26b-a4b" --endpoint "http://localhost:11434" "Hello"
```

### フラグ

```
入力:
  -p, --prompt              プロンプトテキスト
  -f, --file                入力ファイルパス（- で stdin）
  -s, --system-prompt       システムプロンプトテキスト
  -S, --system-prompt-file  システムプロンプトファイルパス
  -i, --image               画像ファイルパス（複数指定可、順序保持）

モデル / エンドポイント:
  -m, --model               モデル名
      --endpoint            API base URL

実行モード:
      --stream              ストリーミング出力
      --batch               行単位バッチ処理

出力フォーマット:
      --format              text（デフォルト）| json | jsonl
      --json-schema         JSONスキーマファイルパス

セキュリティ:
      --no-safe-input       プロンプトインジェクション防御を無効化
  -q, --quiet               警告抑制
      --debug               デバッグ出力

設定:
  -c, --config              設定ファイルパス
```

### 制約事項

- `--stream` と `--batch` は排他
- `--format jsonl` は `--batch` が必要
- `--json-schema` と `--stream` は非互換
- `--image` と `--batch` は非互換
- 対応画像フォーマット: JPEG (`.jpg`, `.jpeg`)、PNG (`.png`)

### データ隔離

パイプ入力（stdin）やファイル入力（`-f`）はデフォルトでノンスタグXMLラッピングされ、
プロンプトインジェクションを防御します。システムプロンプト内で `{{DATA_TAG}}` を使用して
タグ名を参照可能。`--no-safe-input` で無効化。

## ビルド

```bash
make build      # dist/llm-cli（現在のプラットフォーム）
make build-all  # 全プラットフォーム（linux/darwin/windows, amd64/arm64）
make test       # テスト実行
make check      # vet + test + build
```

## ドキュメント

- [アーキテクチャ](docs/ja/architecture.ja.md) — 設計判断とデータフロー
- [構造化出力ガイド](docs/ja/structured-output.ja.md) — JSON スキーマ、フォールバック戦略
- [RFP](docs/ja/llm-cli-rfp.ja.md) — 要件定義・企画ドキュメント

## ライセンス

MIT License。詳細は [LICENSE](LICENSE) を参照。
