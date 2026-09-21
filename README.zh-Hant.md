# Codex Selective Ping（CPA 插件）

[English](README.md) | [繁體中文](README.zh-Hant.md) | [日本語](README.ja.md)

獨立的 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 插件。靈感來自 [`cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping)，但只 ping `accounts` 白名單裡的帳號。`accounts` 為空時，誰都不 ping。

- 插件 ID：`codex-selective-ping`
- 固定模型：`gpt-5.6-luna`
- 設定持久化：宿主 `plugins.configs.codex-selective-ping`
- 管理 UI：繁體中文／英文／日文（預設對齊 CPA 管理中心語系）

## 安裝

### 1. 插件商店（建議）

在 CPA 管理中心新增自訂插件來源，再安裝 **Codex Selective Ping**：

```text
https://raw.githubusercontent.com/danielhuang-030/cpa-plugin-codex-selective-ping/main/registry.json
```

把動態庫放到 CPA 插件目錄（常見為 `plugins/`），或讓商店把 release zip 解到該處：

```text
codex-selective-ping_0.1.3_linux_amd64.zip
└── codex-selective-ping.so
```

### 2. 設定 CPA

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

- `accounts` 可與 `auth_index`（精確）或 email／name／account（不分大小寫）匹配。
- 空的 `accounts` → 排程與手動執行的 attempted 皆為 0。
- CPA 啟動時不會立刻 ping，等到下一個設定時段才跑。

必要時重啟或重載 CPA，然後開啟：

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

### 3. 手動安裝（可選）

若不用商店：

```bash
git clone https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping.git
cd cpa-plugin-codex-selective-ping
docker compose up -d --build
docker compose exec -T dev make build-linux
cp package/codex-selective-ping.so /path/to/cpa/plugins/
```

套用上方 YAML 後重啟 CPA。

## 管理

API：

```text
GET  /v0/management/plugins/codex-selective-ping/status
POST /v0/management/plugins/codex-selective-ping/run
GET/PATCH /v0/management/plugins/codex-selective-ping/config
```

- `POST .../run` 回 202；若已在執行則 409。
- 排程關閉（`schedule_enabled: false`）時，「立刻執行」仍可用，只是每日排程會停。
- 資源頁語系跟隨 CPA 管理中心（`cli-proxy-language`／`Accept-Language`）。可用 `?lang=zh-Hant|en|ja` 覆蓋。不支援的語系退回繁體中文。本插件不會寫入 CPA 的語系鍵。
- UI 主題跟隨 CPA（`cli-proxy-theme` 等），可用 `?theme=light|dark` 覆寫；套用 `data-theme` 於 `<html>`。
- 額度欄（Plan／5h／週限）只顯示宿主提供的值；沒有就顯示「—」。

## 參考

排程、ping 與宿主 ABI 語意對齊 [`jiz4oh/cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping)。本專案獨立；差異在帳號白名單、管理介面，以及設定如何持久化。

## 建置與開發（次要）

給貢獻者看。若用商店安裝，可略過。

```bash
docker compose up -d --build
docker compose exec -T dev go test ./... -count=1
docker compose exec -T dev make build-linux
# 產出 package/codex-selective-ping.so
```

本機：

```bash
make test
make build-linux
```

macOS：

```bash
CGO_ENABLED=1 go build -buildmode=c-shared -o codex-selective-ping.dylib .
```

## 發版

CPA 插件商店在**未安裝**時只顯示 `registry.json` 的 `version`（不會查 GitHub latest；已安裝才會查並判斷是否可更新）。發版時請同步更新：

1. `Makefile` 的 `VERSION`
2. `registration.go` 的 `version`
3. `registry.json` 的 `plugins[0].version`

打 tag 前執行 `make verify-version`。
