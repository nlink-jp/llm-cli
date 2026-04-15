# アーキテクチャ: llm-cli

> 最終更新: 2026-04-16

## このツールが存在する理由

本ツールは [lite-llm](https://github.com/nlink-jp/lite-llm)（アーカイブ済）の後継。
lite-llm は有能な OpenAI 互換 CLI だったが、共有ライブラリ
[nlk](https://github.com/nlink-jp/nlk) より前に設計された。そのため、プロンプト
インジェクション防御、JSON 修復、thinking タグ除去の独自実装を抱えており、
nlk が提供するテスト済みの再利用可能パッケージと重複していた。

llm-cli はゼロから構築し、以下を実現した:

- **重複コードの排除** — 独自の isolation、JSON 抽出、タグ除去を nlk/guard、
  nlk/jsonfix、nlk/strip に置き換え
- **VLM 対応の追加** — VLM モデル向けマルチイメージ入力（`-i`）。
  lite-llm には無かった機能
- **リトライ耐性の追加** — nlk/backoff による一時的 API エラーへの対応
- **cli-series への配置** — lite-llm は lite-series だったが、llm-cli は
  LLM API エンドポイントのサービスクライアントとして cli-series に配置

## nlk 統合を選んだ理由（独自実装ではなく）

lite-llm が独自に実装していたもの:

| 関心事 | lite-llm の実装 | llm-cli での置き換え |
|--------|-----------------|---------------------|
| プロンプトインジェクション防御 | `internal/isolation/`（3バイトノンス） | nlk/guard（128ビットノンス） |
| JSON 抽出・修復 | `internal/output/`（アドホックパース） | nlk/jsonfix（再帰下降パーサー） |
| thinking タグ除去 | `internal/output/`（インライン除去） | nlk/strip（正規表現、マルチフォーマット） |
| エラー時リトライ | なし | nlk/backoff（指数バックオフ + ジッター） |
| 出力バリデーション | なし | nlk/validate（ルールベース） |

統合のメリット:

- **より強力なセキュリティ** — nlk/guard は 128 ビットノンス使用（lite-llm は 24 ビット）。
  ErrTagCollision による多層防御
- **より良い JSON 修復** — nlk/jsonfix はシングルクォート、末尾カンマ、コメント、
  クォートなしキー、エスケープ済み JSON に対応
- **単一の信頼源** — nlk のバグ修正が全利用ツールに自動反映

## マルチモーダルメッセージを選んだ理由（独立画像 API ではなく）

OpenAI の chat/completions API は、ユーザーメッセージの `content` 配列に
base64 データ URI として画像を埋め込むマルチモーダル入力をサポートしている:

| 代替案 | 却下理由 |
|--------|----------|
| 別の `/v1/images` エンドポイント | OpenAI chat API に存在しない。LM Studio/Ollama にもない |
| アップロード後に参照 | 状態管理が必要。ローカル LLM はファイルアップロード未対応 |
| URL ベースの画像参照 | LLM サーバーに URL フェッチが必要。ローカルファイルでは不安定 |

base64 インラインが LM Studio、Ollama、リモート OpenAI のすべてで一貫して動作する
唯一のアプローチ。サーバー側フェッチ不要、状態不要、追加エンドポイント不要。

**画像の順序は重要** — 画像は `-i` フラグの順序（左から右）で content 配列に追加される。
「1枚目と2枚目の画像を比較してください」のようなプロンプトは、この順序が決定論的で
あることに依存する。

## response_format_strategy を採用した理由

ローカル LLM サーバーの `response_format` サポートは異なる:

| サーバー | `json_object` | `json_schema` | 非対応時の動作 |
|----------|:---:|:---:|---|
| LM Studio | 対応 | 対応（文法ベース） | 記述的メッセージ付き 400 エラー |
| Ollama | 部分対応 | 無視 | フィールドをサイレントに無視 |
| OpenAI | 対応 | 対応 | N/A（常に対応） |

単一の戦略では全バックエンドに対応できない。3モード方式（lite-llm から継承）で対処:

- **`auto`** — まずネイティブを試行。400/422 で "response_format" や "not supported"
  等のキーワードを検出したら、response_format なしでリトライし、システムプロンプトに
  フォーマット要件を注入。安全なデフォルト。
- **`native`** — 常に `response_format` を送信。API が対応していると分かっている場合に
  使用。非対応ならハードエラー。
- **`prompt`** — `response_format` を送信しない。常にシステムプロンプト経由で注入。
  フィールドをサイレントに無視する Ollama に推奨。

フォールバック検出はエラーボディで以下を確認: `response_format`、`not supported`、
`unsupported`、`unknown field`。

## Go を選んだ理由

- **シングルバイナリ配布** — パイプライン利用に不可欠
  （`echo "query" | llm-cli --format json | jq`）
- **nlk 統合** — nlk は外部依存ゼロの Go ライブラリ
- **cli-series の一貫性** — gem-cli、scli、splunk-cli はすべて Go
- **クロスコンパイル** — `make build-all` で `CGO_ENABLED=0` の5プラットフォーム
  バイナリを生成

## セキュリティ

| 関心事 | 対策 |
|--------|------|
| プロンプトインジェクション | stdin/ファイル入力を nlk/guard ノンスタグ XML でラップ（128ビットエントロピー） |
| タグ衝突 | ErrTagCollision チェックによる多層防御（確率: 2^-128） |
| 認証情報漏洩 | 設定ファイルのパーミッションチェック（`perm & 0077 != 0` で警告） |
| API キー転送 | Authorization ヘッダーの Bearer トークン（HTTPS 推奨） |
| 画像データ漏洩 | 画像は base64 インラインで送信。中間ストレージやキャッシュなし |
| LLM 出力操作 | nlk/strip が JSON 抽出前に thinking タグを除去 |

## データフロー

```
入力ソース                       処理パイプライン                    出力
──────────                       ────────────────                    ────
-p flag  ──┐                     ┌──────────────────┐
positional ─┤ ReadUserInput ──→ │ nlk/guard.Wrap    │──→ APIリクエスト
-f file  ──┤ (ソース追跡)       │ (SourceExternalの場合) │    ┌──────────┐
stdin    ──┘                     └──────────────────┘    │ OpenAI   │
                                                         │ 互換     │
-s flag  ──┐                     ┌──────────────────┐    │ endpoint │
-S file  ──┤ ReadSystemPrompt → │ guard.Expand      │──→ │          │
            └                    │ ({{DATA_TAG}})    │    └────┬─────┘
                                 └──────────────────┘         │
-i paths ──→ LoadImages ──→ base64エンコード ──→ content配列   │
                                                               ▼
                                 ┌──────────────────┐    レスポンス
                                 │ nlk/strip        │←──────┘
                                 │ nlk/jsonfix      │
                                 │ output.Formatter │──→ stdout
                                 └──────────────────┘
```
