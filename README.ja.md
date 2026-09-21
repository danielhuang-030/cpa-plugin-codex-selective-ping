# Codex Selective Ping（CPA プラグイン）

[English](README.md) | [繁體中文](README.zh-Hant.md) | [日本語](README.ja.md)

独立した [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) プラグインです。[`cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping) を参考にしていますが、`accounts` に列挙したアカウントだけを ping します。`accounts` が空なら誰にも ping しません。

- プラグイン ID: `codex-selective-ping`
- 固定モデル: `gpt-5.6-luna`
- 設定の永続化: ホストの `plugins.configs.codex-selective-ping`
- 管理 UI: 繁体字中国語 / 英語 / 日本語（既定は CPA 管理センターの言語に合わせる）

## インストール

### 1. プラグインストア（推奨）

CPA 管理センターで次のカスタムソースを追加し、**Codex Selective Ping** をインストールします：

```text
https://raw.githubusercontent.com/danielhuang-030/cpa-plugin-codex-selective-ping/main/registry.json
```

共有ライブラリを CPA のプラグインディレクトリ（多くは `plugins/`）へ置くか、ストアに release zip を展開させます：

```text
codex-selective-ping_0.1.2_linux_amd64.zip
└── codex-selective-ping.so
```

### 2. CPA を設定

```yaml
plugins:
  enabled: true
  dir: plugins
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
      accounts:
        - "user@example.com"
        - "auth_index_or_name"
```

- `accounts` は `auth_index`（完全一致）または email / name / account（大文字小文字を区別しない）と照合します。
- `accounts` が空 → スケジュール実行も手動実行も attempted は 0 です。
- CPA 起動時には ping せず、次の設定時刻まで待ちます。

必要なら CPA を再起動／再読込し、次を開きます：

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

### 3. 手動インストール（任意）

ストアを使わない場合：

```bash
git clone https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping.git
cd cpa-plugin-codex-selective-ping
docker compose up -d --build
docker compose exec -T dev make build-linux
cp package/codex-selective-ping.so /path/to/cpa/plugins/
```

上記 YAML を適用して CPA を再起動してください。

## 管理

API:

```text
GET  /v0/management/plugins/codex-selective-ping/status
POST /v0/management/plugins/codex-selective-ping/run
GET/PATCH /v0/management/plugins/codex-selective-ping/config
```

- `POST .../run` は 202。実行中なら 409 です。
- スケジュールが無効（`schedule_enabled: false`）でも **今すぐ実行** は使えます。止まるのは日次スケジュールだけです。
- リソースページの言語は CPA 管理センター（`cli-proxy-language` / `Accept-Language`）に従います。`?lang=zh-Hant|en|ja` で上書きできます。未対応ロケールは繁体字中国語に戻します。本プラグインは CPA の言語キーを書き込みません。
- UI テーマは CPA（`cli-proxy-theme` など）に追従。`?theme=light|dark` で上書き可。`<html>` に `data-theme` を設定。
- クォータ列（Plan / 5h / 週次）はホスト提供値のみ。無い場合は「—」。

## 参考

スケジュール、ping、ホスト ABI の意味は [`jiz4oh/cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping) に揃えています。本プロジェクトは独立しており、違いはアカウント許可リスト、管理 UI、設定の保存方法です。

## ビルドと開発（二次情報）

コントリビューター向け。ストアから入れる場合はスキップして構いません。

```bash
docker compose up -d --build
docker compose exec -T dev go test ./... -count=1
docker compose exec -T dev make build-linux
# package/codex-selective-ping.so を出力
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
