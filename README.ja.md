# Codex Selective Ping（CPA プラグイン）

[English](README.md) | [繁體中文](README.zh-Hant.md) | [日本語](README.ja.md)

独立した [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) プラグインです。[`cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping) を参考にしていますが、`accounts` に列挙したアカウントだけを ping します。`accounts` が空なら誰にも ping しません。

- プラグイン ID: `codex-selective-ping`
- 固定モデル: `gpt-5.6-luna`
- 設定の永続化: ホストの `plugins.configs.codex-selective-ping`
- 管理 UI: 繁体字中国語 / 英語 / 日本語（既定は CPA 管理センターの言語に合わせる）

## 参考

本プロジェクトは新規の独立プラグインです。スケジュール、ping、ホスト ABI の意味は [`jiz4oh/cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping) に揃えています。違いはアカウント許可リスト、管理 UI、設定の保存方法です。

## CPA 設定

```yaml
plugins:
  enabled: true
  dir: plugins
  configs:
    codex-selective-ping:
      enabled: true
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

意味:

- `accounts` は `auth_index`（完全一致）または email / name / account（大文字小文字を区別しない）と照合します。
- `accounts` が空 → スケジュール実行も手動実行も attempted は 0 です。
- CPA 起動時には ping せず、次の設定時刻まで待ちます。

## 管理

リソースページ:

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

API:

```text
GET  /v0/management/plugins/codex-selective-ping/status
POST /v0/management/plugins/codex-selective-ping/run
```

UI からの保存はホスト経由です:

```text
GET/PATCH /v0/management/plugins/codex-selective-ping/config
```

`POST .../run` は 202 を返します。実行中なら 409 です。スケジュールが無効（`enabled: false`）でも **今すぐ実行** は使えます。止まるのは日次スケジュールだけです。

リソースページの言語は CPA 管理センター（`cli-proxy-language` / `Accept-Language`）に従います。`?lang=zh-Hant|en|ja` で上書きできます。未対応のロケールは繁体字中国語に戻します。本プラグインは CPA の言語キーを書き込みません。

クォータ列（Plan / 5h / 週次）はホストが提供した値だけを表示し、無い場合は「—」です。

## ビルド

Docker Compose（Go 1.24 + gcc/CGO）を推奨します:

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

共有ライブラリを CPA のプラグインディレクトリへコピーしてください。

## 開発

```bash
docker compose exec -T dev go test ./...
```
