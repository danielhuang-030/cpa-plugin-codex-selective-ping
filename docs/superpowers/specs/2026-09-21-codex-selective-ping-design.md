# Codex Selective Ping（CPA 插件）設計規格

日期：2026-09-21  
狀態：已核准（UI mock + spec OK）  
參考：`jiz4oh/cpa-plugin-codex-auto-ping`、CLIProxyAPI Plugin Development 文件

## 1. 目標

做一支**全新獨立**的 CLIProxyAPI（CPA）動態庫插件，行為接近 `codex-auto-ping`，但：

1. 可**指定**要 ping 的 Codex 帳號（非全發）
2. 提供**可用的單頁管理 UI**（勾選帳號、編排程、儲存、立刻執行、看結果）
3. 設定以宿主 `plugins.configs` 為唯一真相；UI 經 Management API 持久化

成功標準：

- 空的 `accounts` → 排程與手動 Run 都不 ping（attempted=0，明確提示）
- 只 ping 與白名單匹配的 Codex 帳號
- 管理頁可不改 YAML 手改帳號清單（透過 PATCH config）
- 能建出平台對應的 c-shared 產物（`.so` / `.dylib` / `.dll`）

非目標：

- 改 CPA 核心排程／帳號路由
- 自訂模型（沿用參考插件固定模型語意）
- 寫入 auth 檔或保存 OAuth token
- 對 upstream `codex-auto-ping` 開 PR（採獨立插件路線 A）

## 2. 架構與元件

插件 ID（暫定）：`codex-selective-ping`

| 元件 | 職責 |
|------|------|
| Config | 解析／驗證宿主傳來的 YAML：`enabled`、`timezone`、`times`、`accounts` |
| Scheduler | 依時區與每日時段觸發；啟動時不立刻全 ping |
| Selector | 從 `host.auth.list` 取 Codex 帳號，只保留與 `accounts` 匹配者 |
| Pinger | `host.auth.get` + `host.http.do` 發固定模型小請求；視窗／重試語意對齊 auto-ping |
| RunState | 記憶體：是否 running、上次摘要、每帳號結果（不存 token） |
| Management | Resource 單頁 UI；`GET status`、`POST run`；儲存走宿主 config API |

資料流：

```
排程 / Run Now
  → Selector(accounts)
  → Pinger(selected)
  → RunState
  → GET status / Resource 頁刷新

UI 儲存
  → Management Key
  → PATCH /v0/management/plugins/codex-selective-ping/config
  → 宿主持久化 YAML
  → plugin.reconfigure
  → 套用新 Config
```

## 3. 設定模型

宿主設定範例：

```yaml
plugins:
  enabled: true
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

- `accounts`：字串列表；可與 `auth_index`／email／name／account 匹配（email／name 大小寫不敏感）
- **空列表 = 不 ping任何人**（最安全預設；不做「空=全發」）
- 不提供隱含 `ping_all`；若未來需要全發，應顯式設計，本規格不做

持久化（官方建議）：

- 使用者設定只存在 `plugins.configs.codex-selective-ping`
- UI 使用 `GET`／`PUT`／`PATCH /v0/management/plugins/{id}/config`
- 私有 state 檔（若有）只用於 run 狀態／log，不存 token、不取代設定

## 4. UI 與 API

### 4.1 Resource 單頁（繁中為主）

區塊：

1. 概況：啟用、版本、模型、時區、時段、下次執行、是否執行中
2. 排程：時區、時段增刪
3. 帳號：列表 + checkbox；全選／清除；未勾選明確提示不會 ping
   - **額度欄（本規格增補）**：盡可能顯示 CPA 已提供的 Codex **5h** 與 **週限** 資訊（剩餘／用量、reset／到期時間、plan 若有）
   - 資料來源優先序：宿主 Management `auth-files`／runtime 已暴露的 quota 欄位（如 PR #3068／後續 observe quota）；若插件 ABI 有 `host.auth.get_runtime` 則一併讀取
   - **有就顯示、沒有就顯示「—」**；不自行推算或捏造剩餘量
   - 額度欄為唯讀資訊，不影響 Selector（選不選仍只看 `accounts` 白名單）；排程仍依本插件規則，不因 UI 顯示而改 CPA 冷卻邏輯
4. 動作：Management Key、儲存設定、立刻執行、最近結果表

刻意不做：無 Key 的資源頁直接改設定；不把 Key 寫入 `localStorage`（當次輸入；若管理中心同源已有 Key 可選用，不強制）。

### 4.2 插件 Management 路由

- `GET .../status`：JSON（發現的帳號含 `selected`、last run、排程狀態）
- `POST .../run`：非同步接受（202）；僅 ping 目前 config 選中帳號；已在跑 → 409

### 4.3 宿主 Config API（儲存）

- UI 帶 Management Key 呼叫宿主 `GET/PATCH /v0/management/plugins/codex-selective-ping/config`
- 寫入 `accounts`／`timezone`／`times`／`enabled` 等
- 敏感操作不放在未認證的 resource GET

## 5. 錯誤處理與邊界

| 情況 | 行為 |
|------|------|
| `accounts` 空或全不符 | 結束但 attempted=0；UI 顯示未選／已跳過 |
| 單帳 token／disabled 失敗 | 該帳失敗，其餘繼續；彙總 failed |
| 已在 running | `POST /run` → 409 |
| 壞 timezone／HH:MM | config 驗證 4xx，不套用 |
| 上游 429／限額 | 記 limited（對齊 auto-ping），不中斷整批策略與參考一致 |
| Management Key 錯／缺 | 儲存與 Run 失敗並提示，不靜默 |

## 6. 測試策略（TDD）

1. Selector 純函式：空→0；email／auth_index／name；大小寫；非 Codex 排除
2. Config parse／validate：times、timezone、accounts
3. Run 編排（mock host）：只 ping 選中；running 互斥
4. Management handler：status 形狀、run 202／409
5. 建置：`c-shared` 產出動態庫

## 7. 實作與交付邊界

- 語言：Go + CGO c-shared（對齊 CPA 插件 ABI）
- 盡量重用 auto-ping 的 host 呼叫與 ping／排程語意，差異集中在 Selector、Config、UI／Management
- 交付：原始碼、`README`、`registry.json`、建置說明、design spec、之後的 implementation plan
- 本機開發於 box（不依賴 Cloud agent）

## 8. 決策紀錄

- 全新獨立插件（非改 upstream、非薄封裝）
- 未選帳號 → 不 ping
- 儲存：宿主 `plugins.configs` + Management API；非僅記憶體
- 整體做法 A：獨立插件 + 單頁 UI
- 帳號列顯示 CPA 可見的 5h／週限（有則顯示、無則 —；不捏造）
