# Codex Selective Ping

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/danielhuang-030/cpa-plugin-codex-selective-ping)](https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping/releases)

[English](README.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md)

[CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)（CPA）プラグイン。許可リストのアカウントだけに小さな本物の Codex リクエストを送ります。[`cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping) を参考にしており、`accounts` が空なら誰にも ping しません。

## 機能

- **許可リストのみ** — `accounts` に従う。空配列 → attempted は 0
- **固定モデル** — `gpt-5.6-luna`（変更不可）
- **日次スケジュール** — IANA タイムゾーン + `HH:MM`。CPA 起動時には ping しない
- **管理 UI** — 繁体字中国語 / 英語 / 日本語（CPA 管理センターに追従。`?lang=` / `?theme=` で上書き可）
- **直近実行の永続化** — `{CPA ルート}/data/codex-selective-ping/run_history.json`（`auth-dir` / `auths/` には書かない）
- **プラグイン ID** — `codex-selective-ping` · 設定キー `plugins.configs.codex-selective-ping`

## 要件

- 動的プラグインが有効な [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)
- ストア / 配布 `.so` は Linux `amd64`（他環境は [開発](#開発)）

## インストール

### 1. プラグインストアのソースを追加

CPA のメイン設定（多くはバイナリ横の `config.yaml`、またはデプロイでマウントしたパス）を編集します。トップレベル `plugins:` の `store-sources` に次の URL を**追加**します。公式レジストリは残ります。

```yaml
plugins:
  enabled: true
  dir: plugins
  store-sources:
    - "https://raw.githubusercontent.com/danielhuang-030/cpa-plugin-codex-selective-ping/main/registry.json"
```

注意:

| こうする | こうしない |
| --- | --- |
| `store-sources` を `enabled` / `dir` / `configs` と**同じ階層**に置く | `plugins.configs` の中に入れる |
| `port` や `auth-dir` と同じ `config.yaml` に書く | プラグイン専用の別ファイルにする |
| 既存の `store-sources` があれば配列に 1 件追加 | 意図なくリスト全体を上書きする |

保存後に **CPA を再起動**します。管理センター → プラグインストアから **Codex Selective Ping** をインストールします。

ストアは GitHub Release の zip をプラグインディレクトリ（多くは `plugins/`）へ展開します:

```text
codex-selective-ping_0.1.7_linux_amd64.zip
└── codex-selective-ping.so
```

各 Release にはファイル名が正確に `checksums.txt` のアセットが必要です（CPA はこの名前だけを探します）。

### 2. プラグインを設定

同じ `config.yaml` の `plugins.configs` に、キー名が正確に `codex-selective-ping` のブロックを追加します:

```yaml
plugins:
  enabled: true
  dir: plugins
  store-sources:
    - "https://raw.githubusercontent.com/danielhuang-030/cpa-plugin-codex-selective-ping/main/registry.json"
  configs:
    codex-selective-ping:
      enabled: true
      schedule_enabled: true
      timezone: Asia/Taipei
      times:
        - "06:00"
        - "11:00"
        - "16:00"
        - "21:00"
      history_limit: 60
      retry_count: 2
      accounts:
        - "user@example.com"
        - "auth_index_or_name"
```

必要なら CPA を再起動／再読込し、ステータスページを開きます:

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

### 3. 手動インストール（任意）

ストアを使わず共有ライブラリを自分でビルドする場合:

```bash
git clone https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping.git
cd cpa-plugin-codex-selective-ping
docker compose up -d --build
docker compose exec -T dev make build-linux
cp package/codex-selective-ping.so /path/to/cpa/plugins/
```

上記の `plugins.configs.codex-selective-ping` を適用して CPA を再起動してください。

## 設定リファレンス

| キー | 型 | 説明 |
| --- | --- | --- |
| `enabled` | bool | ホスト上のプラグインインスタンスのオン／オフ（CPA ライフサイクル） |
| `schedule_enabled` | bool | 日次スケジュールのオン／オフ。`false` でも **今すぐ実行** は使える |
| `timezone` | string | IANA タイムゾーン（例: `Asia/Taipei`） |
| `times` | string[] | 1 つ以上の `HH:MM` |
| `accounts` | string[] | 許可リスト。`auth_index`（完全一致）または email / name / account（大文字小文字無視）。空 → ping 0 回 |
| `data_dir` | string | 任意。`run_history.json` のディレクトリ（相対パスは CPA cwd 基準） |
| `history_limit` | int | 保持する実行履歴の上限（既定 **60**；≤0 は 60） |
| `retry_count` | int | **limited**／クォータ失敗時に、何回追加で再試行するか。間隔は固定 **60 秒**（既定 **2**；`0` で無効）。limited 以外はこの経路で再試行しません。 |
| `state_path` | string | 任意。直近実行ファイルのフルパス（`data_dir` より優先） |

既定の直近実行パス: `{CPA ルート}/data/codex-selective-ping/run_history.json`（`plugins/` の親）。

実行履歴は新しい順で `run_history.json` に保存します（既定は `{CPA ルート}/data/codex-selective-ping/`）。同じディレクトリに旧形式の単一オブジェクト `last_run.json` だけがある場合は一度移行し、旧ファイルを削除します。

## 管理

### ステータスページ

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

- 言語: CPA 管理センター（`cli-proxy-language` / `Accept-Language`）。`?lang=zh-Hant|en|ja` で上書き（未対応は繁体字中国語）。本プラグインは CPA の言語キーを書きません。
- テーマ: CPA に追従。`?theme=light|dark` で上書き（`<html data-theme>`）。
- クォータ列（Plan / 5h / 週次）: Plan は Codex id_token（`chatgpt_plan_type`）。Management Key 入力後は CPA の `auth-files` + `api-call`（管理画面と同じ `wham/usage`）で 5h／週次を補完。無い場合は「—」（推測しない）。クォータは参考情報で、ping 対象には影響しません。

### Management API

```text
GET        /v0/management/plugins/codex-selective-ping/status
POST       /v0/management/plugins/codex-selective-ping/run
GET/PATCH  /v0/management/plugins/codex-selective-ping/config
```

`POST .../run` は **202**。実行中なら **409**。

## 開発

コントリビューター向け。ストアから入れる場合はスキップして構いません。

```bash
docker compose up -d --build
docker compose exec -T dev go test ./... -count=1
docker compose exec -T dev make build-linux
# → package/codex-selective-ping.so
```

ローカル:

```bash
make test
make build-linux
```

macOS:

```bash
CGO_ENABLED=1 go build -buildmode=c-shared -o codex-selective-ping.dylib .
```

## リリース

CPA プラグインストアは**未インストール**時、`registry.json` の `version` だけを表示します（GitHub latest はインストール後）。リリース時は次を揃えてください:

1. `Makefile` の `VERSION`
2. `registration.go` の `version`
3. `registry.json` の `plugins[0].version`

タグ付け前に `make verify-version` を実行。各 GitHub Release には次が必要です:

- `codex-selective-ping_<version>_<goos>_<goarch>.zip`
- `checksums.txt`（ファイル名は正確にこれ。`checksums-<version>.txt` だけでは不可）

## 関連プロジェクト

- [`jiz4oh/cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping) — スケジュール／ping／ホスト ABI の参考。本プロジェクトの差分は許可リスト、管理 UI、設定の永続化
- [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) — ホスト
- [CLIProxyAPI Plugins Store](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store) — 公式 registry 形式

## ライセンス

[MIT](LICENSE)
