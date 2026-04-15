# RFP: llm-cli

> Generated: 2026-04-16
> Status: Draft

## 1. Problem Statement

ローカルLLM（LM Studio / Ollama等、OpenAI API互換エンドポイント）をCLIから利用するツール。
現行のlite-llmの新世代後継として、共通ライブラリnlkを全面活用した再設計を行う。

lite-llmは引き続き利用可能だが、今後のメンテナンスはllm-cliに集約する。
主な強化点は、VLMモデルへのマルチイメージ入力対応、gem-cli相当のJSONスキーマによる構造化出力、
およびnlkライブラリ（guard / jsonfix / strip / backoff / validate）への統合である。

パイプフレンドリーなUNIX CLIとして、nlink-jpの他ツールとのパイプライン連携を前提とする。

**ターゲットユーザー:** 自分自身（nlink-jpツール群との連携利用）

## 2. Functional Specification

### Commands / API Surface

```
llm-cli [flags] [prompt]

# 入力
  -p, --prompt              プロンプトテキスト
  -f, --file                入力ファイル or stdin (-)
  -s, --system-prompt       システムプロンプトテキスト
  -S, --system-prompt-file  システムプロンプトファイル
  -i, --image               画像ファイルパス（複数指定可、左から順序保持）

# モデル / エンドポイント
  -m, --model               モデル名
  --endpoint                API base URL

# 実行モード
  --stream                  ストリーミング出力
  --batch                   行単位バッチ処理

# 出力フォーマット
  --format                  text (default) | json | jsonl
  --json-schema             JSONスキーマファイルパス

# セキュリティ
  --no-safe-input           データ隔離（プロンプトインジェクション防御）無効化
  -q, --quiet               警告抑制
  --debug                   デバッグ出力

# 設定
  -c, --config              設定ファイルパス
```

**イメージ入力:**
- `-i image1.png -i image2.jpg` の形式で複数枚を順序付きで指定
- base64エンコードでOpenAI API互換形式（content配列 + image_url + data URI）で送信
- 対応フォーマット: JPEG, PNG（GIF, WebPは実装難易度次第で追加検討）
- バッチモード（`--batch`）との併用は非対応

### Input / Output

**入力:**
- プロンプト: 位置引数 / `-p` フラグ / stdin / `-f` ファイル
- システムプロンプト: `-s` テキスト / `-S` ファイル
- 画像: `-i` ファイルパス（複数可）
- バッチモード: stdinから1行ずつ読み取り

**出力:**
- `--format text`: プレーンテキスト（デフォルト）
- `--format json`: JSON出力（nlk/jsonfixで修復）
- `--format jsonl`: バッチモード専用 `{"input":"...","output":"...","error":null}`
- ストリーミング: `--stream` でトークン単位出力（`--json-schema` とは非互換）

### Configuration

設定ファイル: `~/.config/llm-cli/config.toml`

```toml
[api]
base_url = "http://localhost:1234/v1"   # LM Studio default
api_key = ""                             # リモートAPI接続時に使用
timeout_seconds = 120
response_format_strategy = "auto"        # auto | native | prompt

[model]
name = "default-model"
```

**優先順位:** CLIフラグ > 環境変数（`LLM_CLI_*`） > config.toml > デフォルト値

### External Dependencies

- **LM Studio** / **Ollama**: OpenAI API互換エンドポイント（主要ターゲット）
- **nlk**: guard, jsonfix, strip, backoff, validate（全5パッケージ）
- **Cobra**: CLIフレームワーク
- リモートOpenAI API: 接続可能だが主目的ではない

## 3. Design Decisions

**言語: Go**
- nlink-jpのCLIツール標準言語
- nlkライブラリ（Go）との直接統合
- lite-llm / gem-cliと同一スタック

**nlkへの全面統合:**
- lite-llmが持つ独自のisolation / JSON修復実装をnlkパッケージに置き換え
- `nlk/guard`: プロンプトインジェクション防御（128bit nonceタグ）
- `nlk/jsonfix`: LLM出力のJSON修復
- `nlk/strip`: thinking/reasoningタグ除去
- `nlk/backoff`: APIエラー時の指数バックオフリトライ
- `nlk/validate`: 構造化出力のルールベースバリデーション

**response_format_strategy踏襲:**
- `auto`: ネイティブ送信を試行、非対応ならプロンプト注入にフォールバック
- `native`: 常にresponse_formatを送信
- `prompt`: 常にシステムプロンプトへの注入で対応
- Ollamaはjson_schema形式を無視するため、autoまたはpromptが実質必須

**スコープ外:**
- チャットモード / セッション管理（lite-llmで有効でなかったため）
- Google Search Grounding（gem-cli固有機能）
- コンテキストキャッシュ

## 4. Development Plan

### Phase 1: Core

- プロジェクトスキャフォールド（Makefile, go.mod, 内部パッケージ構造）
- config.toml パース + 環境変数オーバーライド + CLIフラグ統合
- 単発プロンプト実行（blocking）
- ストリーミング出力（SSE）
- `nlk/guard` によるデータ隔離
- `nlk/strip` + `nlk/jsonfix` による出力後処理パイプライン
- `nlk/backoff` によるAPIエラーリトライ
- 単体テスト一式

### Phase 2: Features

- `-i` マルチイメージ入力（base64, JPEG/PNG）
- `--json-schema` 構造化出力 + response_format_strategy
- `--batch` バッチモード + JSONL出力
- `nlk/validate` による出力バリデーション
- 追加画像フォーマット検討（GIF/WebP）

### Phase 3: Release

- README.md / README.ja.md 作成
- CHANGELOG.md 作成
- E2Eテスト（LM Studio + google/gemma-4-26b-a4b）
- AGENTS.md 作成
- リリース作業（タグ、gh release、cli-seriesサブモジュール登録）

**各フェーズは独立してレビュー可能。**

## 5. Required API Scopes / Permissions

- 外部OAuthスコープ / IAMロール: **なし**
- config.tomlにAPIキーフィールドを用意（リモートAPI接続時のため）
- ローカルLLM（LM Studio / Ollama）は認証不要

## 6. Series Placement

**Series: cli-series**

Reason: サービスのCLIクライアントとしての位置づけ。
LM Studio / OllamaのAPIエンドポイントに対するクライアントツールであり、
cli-series（scli, confl-cli, splunk-cli, gem-cli等）と同列の分類が適切。

## 7. External Platform Constraints

### LM Studio

- **response_format**: `json_schema` 対応済（grammar-based sampling / GGUF: llama.cpp, MLX: Outlines）
- **json_object**: 対応済（過去に非対応の時期あり）
- **Vision**: 対応（base64 data URI + image URL）
- **デフォルトエンドポイント**: `http://localhost:1234/v1`
- **Responses API**: 実験的サポート（v0.3.29+）

### Ollama

- **response_format**: OpenAI形式の `json_schema` は**無視される**（Ollama独自のformat使用）
  - → `response_format_strategy=prompt` フォールバックが実質必須
- **Vision**: base64対応だがOpenAI形式と完全互換ではない
  - content配列内のimage_urlでdata URI形式を使用すれば動作する報告あり
- **OpenAI互換API**: 実験的ステータス、破壊的変更の可能性あり
- **デフォルトエンドポイント**: `http://localhost:11434/v1`

### 対応方針

- `response_format_strategy=auto` のフォールバック機構をlite-llmから踏襲
- Ollama向けにはpromptフォールバックを推奨設定としてドキュメント化
- イメージ送信はOpenAI形式（content配列 + image_url + base64 data URI）で統一
- Ollama側の互換対応に依存し、独自変換レイヤーは設けない

---

## Discussion Log

1. **ツール名・課題定義**: llm-cliとして、ローカルLLM向け新世代CLIの方向性を確認。lite-llmの後継だがlite-llm自体は継続、メンテナンスをllm-cliに寄せる方針。

2. **イメージ対応**: 複数枚対応を確認。`-i path1 -i path2` の形式で順序保持。バッチモードとの併用は非対応とした。

3. **チャットモード**: lite-llmで有効な機能でなかったため、現時点では不要と判断。

4. **nlk統合**: 全5パッケージ（guard, jsonfix, strip, backoff, validate）を使用。backoffはLLMのAPIエラー対応として必要。lite-llmの独自実装はnlkに集約。

5. **response_format_strategy**: lite-llmのauto/native/prompt戦略をそのまま踏襲。調査の結果、Ollamaがjson_schemaを無視するため、この仕組みは引き続き重要。

6. **シリーズ配置**: cli-seriesに決定。LLMサービスへのCLIクライアントという位置づけ。

7. **E2Eテスト**: LM Studio + google/gemma-4-26b-a4bで実施。Phase 3で対応。

8. **画像フォーマット**: JPEG/PNGは必須対応。GIF/WebPは実装難易度次第で検討。

9. **外部API制約調査**: LM Studioはjson_schema・Vision共に対応済。Ollamaはjson_schemaが非互換、Visionは部分的対応。フォールバック機構の重要性を確認。
