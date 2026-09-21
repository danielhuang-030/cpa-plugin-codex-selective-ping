# Codex Selective Ping（CPA プラグイン）

[English](README.md) | [繁體中文](README.zh-Hant.md) | [日本語](README.ja.md)

独立した [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) プラグインです。[`cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping) を参考にしていますが、`accounts` に列挙したアカウントだけを ping します。`accounts` が空なら誰にも ping しません。

- プラグイン ID: `codex-selective-ping`
- 固定モデル: `gpt-5.6-luna`
- 設定の永続化: ホストの `plugins.configs.codex-selective-ping`
- 管理 UI: 繁体字中国語 / 英語 / 日本語（既定は CPA 管理センターの言語に合わせる）
- ライセンス: [MIT](LICENSE)

## インストール

### 1. CPA プラグインストアに追加する

CPA のメイン設定ファイル（多くはバイナリ横の `config.yaml`、またはデプロイでマウントしているパス）を編集します。トップレベルの `plugins:` ブロックに、次の registry URL を `store-sources`（配列）へ**追加**します。公式ストアは常に残り、ここでは追加ソースだけを足します。

```yaml
plugins:
  enabled: true
  dir: plugins
  store-sources:
    - "https://raw.githubusercontent.com/danielhuang-030/cpa-plugin-codex-selective-ping/main/registry.json"
  # configs: ...  # 手順 2 を参照。インストール前は configs を空でも可
```

置き場所の注意:

- `port` や `auth-dir` など他の CPA 設定と**同じ** `config.yaml` に書きます。別ファイルではありません。
- `plugins:` 直下で `enabled` / `dir` / `configs` と**同じ階層**。キー名は `store-sources` です。
- `store-sources` を `plugins.configs` の中に入れないでください。
- 既に `store-sources` がある場合は配列に URL を1件追加し、意図なくリスト全体を上書きしないでください。

保存後に **CPA を再起動**し、ストアがソースを読み直すようにします。管理センター → プラグインストアに **Codex Selective Ping** が出たら、そこからインストールします。

ストアは GitHub Release の zip を CPA のプラグインディレクトリ（多くは `plugins/`）へ展開します。現行リリースの例:

```text
codex-selective-ping_0.1.6_linux_amd64.zip
└── codex-selective-ping.so
```

Release アセットには、ファイル名が正確に `checksums.txt` のファイルが必要です（CPA はこの名前だけを探します）。

### 2. このプラグインを設定する

同じ `config.yaml` のまま、`plugins.configs` の下にキー名が正確に `codex-selective-ping` のブロックを追加（またはマージ）します:

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
      accounts:
        - "user@example.com"
        - "auth_index_or_name"
```

- `accounts` は `auth_index`（完全一致）または email / name / account（大文字小文字を区別しない）と照合します。
- `accounts` が空 → スケジュール実行も手動実行も attempted は 0 です。
- CPA 起動時には ping せず、次の設定時刻まで待ちます。
- 直近の実行サマリーは `{CPA ルート}/data/codex-selective-ping/last_run.json`（`plugins/` の親）に保存され、再読込後も管理 UI に残ります。任意の上書き: `data_dir` / `state_path`（相対パスは CPA の cwd 基準）。`auth-dir` / `auths/` には書き込みません。

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

上記の `plugins.configs.codex-selective-ping` YAML を適用して CPA を再起動してください。

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
- クォータ列（Plan / 5h / 週次）：Plan は Codex id_token（`chatgpt_plan_type`）由来。Management Key 入力後は CPA の `auth-files` + `api-call`（管理画面と同じ `wham/usage`）で 5h／週次を補完します。無い場合は「—」（推測しません）。クォータは参考情報で、ping 対象選定には使いません。

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

## リリース

CPA プラグインストアは**未インストール**時、`registry.json` の `version` だけを表示します（GitHub latest はインストール後に照会）。リリース時は次を揃えてください：

1. `Makefile` の `VERSION`
2. `registration.go` の `version`
3. `registry.json` の `plugins[0].version`

タグ付け前に `make verify-version` を実行。各 GitHub Release には次が必要です:

- `codex-selective-ping_<version>_<goos>_<goarch>.zip`
- `checksums.txt`（ファイル名は正確にこれ。`checksums-<version>.txt` だけでは不可）

## ライセンス

[MIT](LICENSE)
