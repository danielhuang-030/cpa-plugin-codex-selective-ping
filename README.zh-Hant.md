# Codex Selective Ping

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/danielhuang-030/cpa-plugin-codex-selective-ping)](https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping/releases)

[English](README.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md)

[CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)（CPA）插件：只對你列入白名單的帳號送出小型真實 Codex 請求。靈感來自 [`cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping)；`accounts` 為空時，誰都不 ping。

## 功能

- **只 ping 白名單** — 依 `accounts`；空陣列 → attempted 為 0
- **可選模型** — 設定欄位 `model` 可選；空字串／省略 → 後備 `gpt-6-luna`。管理 UI 透過 plugin `GET …/models` 代打載入選項（可用「更新模型」；Management Key → `api-keys` → `/v1/models`），不再由瀏覽器直打 `GET /v1/models`
- **每日排程** — IANA 時區 + 全域 `times`；可選每帳號 `account_times`；CPA 啟動時不會立刻 ping
- **管理 UI** — 繁中／英文／日文（跟隨 CPA 管理中心；可用 `?lang=` / `?theme=` 覆寫）；每帳號繼承／自訂排程；可展開的依帳號執行歷史
- **保留上次執行** — `{CPA 根目錄}/data/codex-selective-ping/run_history.json`（不會寫入 `auth-dir` / `auths/`）
- **插件 ID** — `codex-selective-ping` · 設定鍵 `plugins.configs.codex-selective-ping`

## 需求

- 已啟用動態插件的 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)
- 商店／預編譯 `.so` 安裝需 Linux `amd64`（其他平台見 [開發](#開發)）

## 安裝

### 1. 加入插件商店來源

編輯 CPA 主設定檔（通常是二進位旁的 `config.yaml`，或部署時掛載的路徑）。在頂層 `plugins:` 下，把下列 URL **加進** `store-sources`。官方商店會保留；這裡只多一個自訂來源。

```yaml
plugins:
  enabled: true
  dir: plugins
  store-sources:
    - "https://raw.githubusercontent.com/danielhuang-030/cpa-plugin-codex-selective-ping/main/registry.json"
```

注意：

| 要這樣 | 不要這樣 |
| --- | --- |
| `store-sources` 與 `enabled` / `dir` / `configs` **同層** | 寫進 `plugins.configs` 裡 |
| 與 `port`、`auth-dir` 等寫在**同一個** `config.yaml` | 另開「只有插件」的設定檔 |
| 若已有 `store-sources`，在陣列再加一筆 | 整段蓋掉（除非故意拿掉其他來源） |

存檔後**重啟 CPA**。到管理中心 → 插件商店，安裝 **Codex Selective Ping**。

商店會把 GitHub Release zip 解到 CPA 插件目錄（常見為 `plugins/`）：

```text
codex-selective-ping_0.2.8_linux_amd64.zip
└── codex-selective-ping.so
```

每個 Release 必須有檔名剛好為 `checksums.txt` 的資產（CPA 只認這個名字）。

### 2. 設定插件

同一個 `config.yaml`，在 `plugins.configs` 下新增鍵名剛好為 `codex-selective-ping` 的區塊：

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
      model: gpt-6-luna
      accounts:
        - "alice@example.com"
        - "bob@example.com"
      account_times:
        bob@example.com:
          - "07:30"
          - "19:00"
```

此例中 alice 繼承全域四個時段；bob 只用 `07:30` 與 `19:00`。省略 `account_times`（或某帳號清單為空）則全員沿用全域 `times`。

必要時重啟或重載 CPA，然後開啟狀態頁：

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

### 3. 手動安裝（可選）

不用商店時自行編譯：

```bash
git clone https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping.git
cd cpa-plugin-codex-selective-ping
docker compose up -d --build
docker compose exec -T dev make build-linux
cp package/codex-selective-ping.so /path/to/cpa/plugins/
```

再套用上方的 `plugins.configs.codex-selective-ping` 並重啟 CPA。

## 設定參考

| 鍵 | 型別 | 說明 |
| --- | --- | --- |
| `enabled` | bool | 宿主插件實例開關（CPA 生命週期） |
| `schedule_enabled` | bool | 排程總開關。`false` 會停止**所有**排程 ping（含每帳號自訂時段）；「立刻執行」仍可用 |
| `timezone` | string | 全域 IANA 時區（如 `Asia/Taipei`）；不可依帳號分開設定 |
| `times` | string[] | 統一的每日預設 `HH:MM`（帳號繼承時使用） |
| `accounts` | string[] | 白名單：`auth_index`（精確）或 email／name／account（不分大小寫）。空 → 0 次 ping |
| `account_times` | map[string][]string | 可選。帳號 id → `HH:MM` 清單。缺漏或空清單 → 繼承全域 `times`。不在 `accounts` 內的鍵會在儲存時清除 |
| `data_dir` | string | 可選。`run_history.json` 所在目錄（相對路徑相對 CPA cwd） |
| `history_limit` | int | 最多保留幾筆執行紀錄（預設 **60**；≤0 視為 60） |
| `retry_count` | int | 若結果為 **limited**／額度不足，再重試幾次，每次間隔 **60 秒**（預設 **2**；`0` 關閉）。非 limited 失敗不走此重試。 |
| `model` | string | 可選。ping 使用的 Codex 模型 id。空字串／省略 → 後備 `gpt-6-luna` |
| `state_path` | string | 可選。上次執行檔完整路徑（優先於 `data_dir`） |

**排程行為：** 排程器等待白名單內各帳號「有效時段」的**聯集**（有非空自訂則用自訂，否則用全域 `times`）。每次觸發某個 `HH:MM` 時，只 ping 有效時段包含該時段的帳號。手動／「立刻執行」仍針對整份白名單，並寫入歷史（`force`）。

預設上次執行路徑：`{CPA 根目錄}/data/codex-selective-ping/run_history.json`（`plugins/` 上一層）。

執行歷史以新到舊寫入 `run_history.json`（預設 `{CPA 根目錄}/data/codex-selective-ping/`）。同目錄若只有舊版單一物件的 `last_run.json`，會遷移一次後刪除舊檔。

## 管理

### 狀態頁

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

- 語系：跟隨 CPA 管理中心（`cli-proxy-language`／`Accept-Language`）；`?lang=zh-Hant|en|ja` 可覆寫（不支援則繁中）。本插件不會寫入 CPA 語系鍵。
- 主題：跟隨 CPA（`cli-proxy-theme` 等），`?theme=light|dark` 可覆寫（`<html data-theme>`）。
- 帳號：勾選白名單；每個帳號可**繼承**全域 `times` 或使用**自訂** `account_times`。帳號區塊**不**顯示 5h／週限額度欄。
- 執行歷史：可展開每筆執行 → 依帳號顯示狀態／嘗試次數／錯誤；包含排程與 force（手動）執行。
- 歷史分頁：瀏覽器會將執行歷史列表分頁（預設 10 筆，可選 10／20／50）；偏好會儲存在 localStorage 的 `csp-hist-page-size`，不在插件設定中。

### Management API

```text
GET        /v0/management/plugins/codex-selective-ping/status
GET        /v0/management/plugins/codex-selective-ping/models
POST       /v0/management/plugins/codex-selective-ping/run
GET/PATCH  /v0/management/plugins/codex-selective-ping/config
```

`GET .../models` 回傳過濾後的 OpenAI 風格 `{data:[{id}]}`（Management Key → `GET /v0/management/api-keys` 取代理 API Key → `/v1/models`；失敗再試第一個 Codex token）。上游全失敗仍 **200**，回傳後備 id（已設定的 `model` + `gpt-6-luna`）與 `warning`。UI「更新模型」在未填 Management Key 時與儲存設定相同，提示需要 Key。

`POST .../run` 回 **202**；若已在執行則 **409**。

## 開發

給貢獻者。用商店安裝的使用者可略過。

```bash
docker compose up -d --build
docker compose exec -T dev go test ./... -count=1
docker compose exec -T dev make build-linux
# → package/codex-selective-ping.so
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

CPA 插件商店在**未安裝**時只顯示 `registry.json` 的 `version`（安裝後才會查 GitHub latest）。發版時請同步更新：

1. `Makefile` 的 `VERSION`
2. `registration.go` 的 `version`
3. `registry.json` 的 `plugins[0].version`

打 tag 前執行 `make verify-version`。每個 GitHub Release 需包含：

- `codex-selective-ping_<version>_<goos>_<goarch>.zip`
- `checksums.txt`（檔名必須剛好是這個；不要只上傳 `checksums-<version>.txt`）

## 相關專案

- [`jiz4oh/cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping) — 排程／ping／宿主 ABI 參考；本專案多了白名單、管理 UI 與設定持久化
- [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) — 宿主
- [CLIProxyAPI Plugins Store](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store) — 官方 registry 格式

## 授權

[MIT](LICENSE)
