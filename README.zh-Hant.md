# Codex Selective Ping（CPA 插件）

[English](README.md) | [繁體中文](README.zh-Hant.md) | [日本語](README.ja.md)

獨立的 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 插件。靈感來自 [`cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping)，但只 ping `accounts` 白名單裡的帳號。`accounts` 為空時，誰都不 ping。

- 插件 ID：`codex-selective-ping`
- 固定模型：`gpt-5.6-luna`
- 設定持久化：宿主 `plugins.configs.codex-selective-ping`
- 管理 UI：繁體中文／英文／日文（預設對齊 CPA 管理中心語系）

## 參考

本專案是全新獨立插件。排程、ping 與宿主 ABI 語意對齊 [`jiz4oh/cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping)；差異在帳號白名單、管理介面，以及設定如何持久化。

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

語意：

- `accounts` 可與 `auth_index`（精確）或 email／name／account（不分大小寫）匹配。
- 空的 `accounts` → 排程與手動執行的 attempted 皆為 0。
- CPA 啟動時不會立刻 ping，等到下一個設定時段才跑。

## 管理

資源頁：

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

API：

```text
GET  /v0/management/plugins/codex-selective-ping/status
POST /v0/management/plugins/codex-selective-ping/run
```

從 UI 儲存設定（走宿主）：

```text
GET/PATCH /v0/management/plugins/codex-selective-ping/config
```

`POST .../run` 回 202；若已在執行則 409。排程關閉（`enabled: false`）時，「立刻執行」仍可用，只是每日排程會停。

資源頁語系跟隨 CPA 管理中心（`cli-proxy-language`／`Accept-Language`）。可用 `?lang=zh-Hant|en|ja` 覆蓋。不支援的語系退回繁體中文。本插件不會寫入 CPA 的語系鍵。

額度欄（Plan／5h／週限）只顯示宿主提供的值；沒有就顯示「—」。

## 建置

建議用 Docker Compose（Go 1.24 + gcc/CGO）：

```bash
docker compose up -d --build
docker compose exec -T dev go test ./... -count=1
docker compose exec -T dev make build-linux
# 產出 package/codex-selective-ping.so
```

或在本機：

```bash
make test
make build-linux
```

macOS：

```bash
CGO_ENABLED=1 go build -buildmode=c-shared -o codex-selective-ping.dylib .
```

把動態庫複製到 CPA 的插件目錄。

## 開發

```bash
docker compose exec -T dev go test ./...
```
