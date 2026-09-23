package management

import (
	"encoding/json"
	"strconv"
	"strings"
)

type Lang string

const (
	LangZhHant Lang = "zh-Hant"
	LangEn     Lang = "en"
	LangJa     Lang = "ja"
)

// NormalizeLang maps CPA / BCP-47 / raw tags (and Zustand JSON) onto our three UI langs.
// zh-CN, ru, and anything unrecognized fall back to zh-Hant.
func NormalizeLang(raw string) Lang {
	s := strings.TrimSpace(raw)
	if s == "" {
		return LangZhHant
	}
	if strings.HasPrefix(s, "{") {
		var wrap struct {
			State struct {
				Language string `json:"language"`
			} `json:"state"`
			Language string `json:"language"`
		}
		if err := json.Unmarshal([]byte(s), &wrap); err == nil {
			if wrap.State.Language != "" {
				s = wrap.State.Language
			} else if wrap.Language != "" {
				s = wrap.Language
			}
		}
	}
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)
	// Strip quality / extras if somehow present
	if i := strings.IndexAny(lower, ";, "); i >= 0 {
		lower = lower[:i]
	}
	switch {
	case lower == "zh-tw" || lower == "zh-hant" || lower == "zh-hk" || lower == "zh-mo" ||
		lower == "zh_tw" || lower == "zh_hant" || lower == "zh_hk" || lower == "zh_mo":
		return LangZhHant
	case lower == "zh-cn" || lower == "zh-hans" || lower == "zh_cn" || lower == "zh_hans" || lower == "zh":
		return LangZhHant
	case lower == "en" || strings.HasPrefix(lower, "en-") || strings.HasPrefix(lower, "en_"):
		return LangEn
	case lower == "ja" || strings.HasPrefix(lower, "ja-") || strings.HasPrefix(lower, "ja_"):
		return LangJa
	default:
		return LangZhHant
	}
}

// ResolveLang picks UI language: explicit query wins, then Accept-Language, else zh-Hant.
func ResolveLang(queryLang, acceptLanguage string) Lang {
	if strings.TrimSpace(queryLang) != "" {
		return NormalizeLang(queryLang)
	}
	if tag := firstAcceptLanguageTag(acceptLanguage); tag != "" {
		return NormalizeLang(tag)
	}
	return LangZhHant
}

func firstAcceptLanguageTag(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	// Take the tag with the highest q-weight (default q=1); skip empty pieces.
	bestTag := ""
	bestQ := -1.0
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tag := part
		q := 1.0
		if semi := strings.Index(part, ";"); semi >= 0 {
			tag = strings.TrimSpace(part[:semi])
			rest := part[semi+1:]
			for _, param := range strings.Split(rest, ";") {
				param = strings.TrimSpace(param)
				if strings.HasPrefix(strings.ToLower(param), "q=") {
					if v, err := strconv.ParseFloat(strings.TrimSpace(param[2:]), 64); err == nil {
						q = v
					}
				}
			}
		}
		if tag != "" && q > bestQ {
			bestQ = q
			bestTag = tag
		}
	}
	return bestTag
}

func T(lang Lang, key string) string {
	if m, ok := catalogs[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if v, ok := catalogs[LangZhHant][key]; ok {
		return v
	}
	return key
}

var catalogs = map[Lang]map[string]string{
	LangZhHant: {
		"title":                  "Codex Selective Ping",
		"subtitle":               "統一節奏當預設；單一帳號可改成自己的時刻。關掉排程＝全部停；手動仍可跑並寫入歷史。",
		"overview":               "概況",
		"rail_now":               "此刻",
		"rail_whitelist":         "白名單",
		"rail_model":             "模型",
		"rail_key":               "當次金鑰",
		"principles_title":       "怎麼算到點？",
		"principles_body":        "系統看所有勾選帳號的「有效時刻」聯集。到點只打該時段有效的帳號；自訂帳號只在自己的時刻被打。",
		"principles_quota":       "僅供參考",
		"principles_persist":     "重載仍在",
		"status":                 "狀態",
		"version_model":          "版本 / 模型",
		"tz_times":               "時區 / 時段",
		"next_run":               "下次執行",
		"running_label":          "執行中",
		"yes":                    "是",
		"no":                     "否",
		"enabled_on":             "已啟用",
		"enabled_off":            "停用",
		"schedule":               "排程",
		"rhythm_title":           "統一節奏",
		"slot_next":              "即將 · 下次",
		"slot_past":              "已過",
		"enable":                 "啟用",
		"timezone":               "時區",
		"add_time":               "新增時段",
		"add":                    "加入",
		"schedule_hint":          "啟動時不會立刻全 ping；到點才跑。空帳號清單 = 誰都不 ping。",
		"accounts":               "帳號",
		"accounts_who_title":     "帳號與時刻",
		"accounts_empty_heading": "還沒有白名單",
		"accounts_empty_title":   "先選要保活的帳號",
		"accounts_empty_body":    "至少勾一個 Codex 帳號並儲存。空白名單時，排程與立刻執行都會略過全部帳號。",
		"accounts_empty_cta":     "去選帳號",
		"filter_all":             "全部 %d",
		"filter_selected":        "已選 %d",
		"filter_abnormal":        "異常 %d",
		"filter_custom":          "自訂時刻 %d",

		"banner_selected":        "目前已勾選 %d / %d 個帳號。未勾選的不會被排程或「立刻執行」打到。",
		"banner_empty":           "帳號白名單為空：排程與立刻執行都不會 ping 任何人。",
		"select_all":             "全選",
		"clear":                  "清除",
		"col_account":            "帳號",
		"col_plan":               "Plan",
		"col_5h":                 "5h",
		"col_weekly":             "週限",
		"col_status":             "狀態",
		"accounts_hint":          "勾選＝白名單。每張卡可「沿用統一」或「自訂時刻」。不再顯示 5h／週限。",
		"actions":                "動作",
		"key_placeholder":        "Management Key（同分頁暫存，關閉分頁即清除）",
		"save":                   "儲存設定",
		"run_now":                "立刻執行",
		"refresh":                "重新整理",
		"refresh_models":         "更新模型",
		"actions_hint":           "儲存：PATCH 宿主 plugins.configs。立刻執行：只 ping 目前已儲存白名單中的帳號（請先儲存勾選）。",
		"last_run":               "上一輪回報",
		"last_run_sub":           "像收據，不是另一張大表。",
		"last_run_receipt_label": "最近一次",
		"last_run_empty_title":   "還沒跑過",
		"last_run_empty_body":    "儲存白名單後按「立刻執行」，或等排程到點。結果會留在 data/codex-selective-ping。",
		"no_last_run":            "尚無執行紀錄",
		"run_history":            "歷史紀錄",
		"run_history_title":      "執行與歷史",
		"run_history_sub":        "最新收據 + 可展開歷史。手動（force）與排程都入史；展開後依帳號看結果。",
		"col_result":             "結果",
		"col_http":               "HTTP",
		"col_detail":             "說明",
		"need_key":               "需要 Management Key",
		"saving":                 "儲存中...",
		"starting":               "啟動中...",
		"run_polling":            "執行中，完成後會自動更新…",
		"run_already":            "已有執行在進行，改為等待完成…",
		"quota_remain":           "剩",
		"quota_used":             "用",
		"quota_reset":            "重置",
		"auth_index_label":       "認證索引",
		"chip_mode":              "模式",
		"chip_ok":                "成功",
		"chip_limited":           "限額",
		"chip_failed":            "失敗",
		"chip_skipped":           "略過",
		"status_success":         "成功",
		"status_limited":         "限額",
		"status_failed":          "失敗",
		"status_skipped":         "略過",
		"rhythm_sub":             "所有「沿用統一」的帳號吃這份時刻。自訂帳號不在這裡改。",
		"slot_next_inherit":      "即將 · 下次（沿用者）",
		"how_title":              "怎麼算到點？",
		"how_body":               "系統看所有勾選帳號的「有效時刻」聯集。到點只打該時段有效的帳號。",
		"how_global_off":         "全域關",
		"how_global_off_v":       "全部排程停",
		"how_manual":             "手動",
		"how_manual_v":           "打完整白名單",
		"accounts_times_sub":     "勾選＝白名單。每張卡可「沿用統一」或「自訂時刻」。不再顯示 5h／週限。",
		"sched_inherit":          "沿用統一",
		"sched_custom":           "自訂",
		"sched_effective":        "有效：",
		"sched_not_selected":     "未在白名單 — 時刻設定儲存時會忽略／prune",
		"sched_custom_empty":     "自訂時刻為空時，儲存後會改回沿用統一節奏。",
		"next_custom_only":       "下次執行 %s（僅自訂帳號；不在統一時間軸上）",
		"sched_add":              "加入",
		"hist_expand":            "展開",
		"hist_collapse":          "收合",
		"hist_attempts":          "attempts",
		"hist_pager_prev":        "上一頁",
		"hist_pager_next":        "下一頁",
		"hist_pager_per_page":    "每頁",
		"mode_force":             "手動執行",
		"mode_scheduled":         "排程",
		"chip_unselected":        "未選",
		"accounts_none_title":    "還沒有 Codex 帳號",
		"accounts_none_body":     "CPA 認證檔出現後，這裡會列出帳號卡。可先設好統一節奏與時區。",
		"accounts_none_cta":      "重新整理帳號",
		"lang_zh":                "繁中",
		"lang_en":                "English",
		"lang_ja":                "日本語",
	},
	LangEn: {
		"title":                  "Codex Selective Ping",
		"subtitle":               "Global rhythm is the default; any account can use its own times. Schedule off stops all timed runs; manual still works and is recorded.",
		"overview":               "Overview",
		"rail_now":               "Now",
		"rail_whitelist":         "Allowlist",
		"rail_model":             "Model",
		"rail_key":               "Session key",
		"principles_title":       "How does the clock fire?",
		"principles_body":        "The scheduler uses the union of effective times for selected accounts. At each slot it pings only accounts whose effective times include that HH:MM.",
		"principles_quota":       "Informational only",
		"principles_persist":     "Survives reload",
		"status":                 "Status",
		"version_model":          "Version / model",
		"tz_times":               "Timezone / slots",
		"next_run":               "Next run",
		"running_label":          "Running",
		"yes":                    "Yes",
		"no":                     "No",
		"enabled_on":             "Enabled",
		"enabled_off":            "Disabled",
		"schedule":               "Schedule",
		"rhythm_title":           "Global rhythm",
		"slot_next":              "Up next",
		"slot_past":              "Past",
		"enable":                 "Enable",
		"timezone":               "Timezone",
		"add_time":               "Add time",
		"add":                    "Add",
		"schedule_hint":          "Startup does not ping everyone; runs only at scheduled times. An empty account list means nobody is pinged.",
		"accounts":               "Accounts",
		"accounts_who_title":     "Accounts & times",
		"accounts_empty_heading": "No allowlist yet",
		"accounts_empty_title":   "Pick accounts to keep alive",
		"accounts_empty_body":    "Select at least one Codex account and save. With an empty allowlist, schedule and Run now skip everyone.",
		"accounts_empty_cta":     "Choose accounts",
		"filter_all":             "All %d",
		"filter_selected":        "Selected %d",
		"filter_abnormal":        "Abnormal %d",
		"filter_custom":          "Custom %d",

		"banner_selected":        "%d / %d accounts selected. Unchecked accounts are skipped by the schedule and Run now.",
		"banner_empty":           "Account allowlist is empty: neither the schedule nor Run now will ping anyone.",
		"select_all":             "Select all",
		"clear":                  "Clear",
		"col_account":            "Account",
		"col_plan":               "Plan",
		"col_5h":                 "5h",
		"col_weekly":             "Weekly",
		"col_status":             "Status",
		"accounts_hint":          "Checked = allowlist. Each card can inherit the global rhythm or use custom times. 5h/weekly columns are gone.",
		"actions":                "Actions",
		"key_placeholder":        "Management Key (kept for this tab only; cleared when the tab closes)",
		"save":                   "Save settings",
		"run_now":                "Run now",
		"refresh":                "Refresh",
		"refresh_models":         "Refresh models",
		"actions_hint":           "Save: PATCH host plugins.configs. Run now: pings only accounts already saved in the allowlist (save your selection first).",
		"last_run":               "Last run report",
		"last_run_sub":           "A receipt, not another dense table.",
		"last_run_receipt_label": "Latest",
		"last_run_empty_title":   "No runs yet",
		"last_run_empty_body":    "Save the allowlist then press Run now, or wait for the schedule. Results persist under data/codex-selective-ping.",
		"no_last_run":            "No runs yet",
		"run_history":            "Run history",
		"run_history_title":      "Run & history",
		"run_history_sub":        "Latest receipt plus expandable history. Manual (force) and scheduled runs are both recorded; expand for per-account detail.",
		"col_result":             "Result",
		"col_http":               "HTTP",
		"col_detail":             "Detail",
		"need_key":               "Management Key required",
		"saving":                 "Saving...",
		"starting":               "Starting...",
		"run_polling":            "Running — page will refresh when finished…",
		"run_already":            "A run is already in progress — waiting for it to finish…",
		"quota_remain":           "left",
		"quota_used":             "used",
		"quota_reset":            "resets",
		"auth_index_label":       "auth_index",
		"chip_mode":              "mode",
		"chip_ok":                "ok",
		"chip_limited":           "limited",
		"chip_failed":            "failed",
		"chip_skipped":           "skipped",
		"status_success":         "success",
		"status_limited":         "limited",
		"status_failed":          "failed",
		"status_skipped":         "skipped",
		"rhythm_sub":             "Accounts set to inherit use these slots. Custom accounts are edited on their cards.",
		"slot_next_inherit":      "Up next (inheritors)",
		"how_title":              "How does the clock fire?",
		"how_body":               "The scheduler waits on the union of effective times for selected accounts, then pings only matching accounts.",
		"how_global_off":         "Global off",
		"how_global_off_v":       "All schedules stop",
		"how_manual":             "Manual",
		"how_manual_v":           "Full allowlist",
		"accounts_times_sub":     "Checked = allowlist. Each card can inherit or use custom times. No 5h/weekly columns.",
		"sched_inherit":          "Inherit",
		"sched_custom":           "Custom",
		"sched_effective":        "Effective:",
		"sched_not_selected":     "Not on allowlist — custom times are ignored/pruned on save",
		"sched_custom_empty":     "Empty custom times inherit the global rhythm on save.",
		"next_custom_only":       "Next run %s (custom accounts only; not on the global timeline)",
		"sched_add":              "Add",
		"hist_expand":            "Expand",
		"hist_collapse":          "Collapse",
		"hist_attempts":          "attempts",
		"hist_pager_prev":        "Previous",
		"hist_pager_next":        "Next",
		"hist_pager_per_page":    "Per page",
		"mode_force":             "Manual run",
		"mode_scheduled":         "Scheduled",
		"chip_unselected":        "Off list",
		"accounts_none_title":    "No Codex accounts yet",
		"accounts_none_body":     "When CPA auth files appear, account cards show here. You can set the global rhythm first.",
		"accounts_none_cta":      "Refresh accounts",
		"lang_zh":                "繁中",
		"lang_en":                "English",
		"lang_ja":                "日本語",
	},
	LangJa: {
		"title":              "Codex Selective Ping",
		"subtitle":           "共通リズムが既定。アカウントごとに独自時刻へ切替可。スケジュールオフで全停止；手動は引き続き実行・履歴に残ります。",
		"overview":           "概要",
		"rail_now":           "いま",
		"rail_whitelist":     "許可リスト",
		"rail_model":         "モデル",
		"rail_key":           "セッションキー",
		"principles_title":   "何時に発火する？",
		"principles_body":    "選択中アカウントの有効時刻の和集合で待ちます。発火時刻に含まれるアカウントだけ ping します。",
		"principles_quota":   "参考情報",
		"principles_persist": "再読込後も残る",

		"status":                 "状態",
		"version_model":          "バージョン / モデル",
		"tz_times":               "タイムゾーン / 時刻",
		"next_run":               "次回実行",
		"running_label":          "実行中",
		"yes":                    "はい",
		"no":                     "いいえ",
		"enabled_on":             "有効",
		"enabled_off":            "無効",
		"schedule":               "スケジュール",
		"rhythm_title":           "共通リズム",
		"slot_next":              "次",
		"slot_past":              "済",
		"enable":                 "有効",
		"timezone":               "タイムゾーン",
		"add_time":               "時刻を追加",
		"add":                    "追加",
		"schedule_hint":          "起動時に全員へ ping しません。予定時刻のみ実行します。アカウント一覧が空なら誰にも ping しません。",
		"accounts":               "アカウント",
		"accounts_who_title":     "アカウントと時刻",
		"accounts_empty_heading": "許可リストが空です",
		"accounts_empty_title":   "まずアカウントを選んでください",
		"accounts_empty_body":    "Codex アカウントを1つ以上選んで保存してください。許可リストが空だと、スケジュールも今すぐ実行も全員スキップします。",
		"accounts_empty_cta":     "アカウントを選ぶ",
		"filter_all":             "すべて %d",
		"filter_selected":        "選択 %d",
		"filter_abnormal":        "異常 %d",
		"filter_custom":          "独自時刻 %d",

		"banner_selected":        "現在 %d / %d アカウントを選択中。未選択はスケジュールと「今すぐ実行」の対象外です。",
		"banner_empty":           "アカウント許可リストが空です。スケジュールも今すぐ実行も誰にも ping しません。",
		"select_all":             "すべて選択",
		"clear":                  "クリア",
		"col_account":            "アカウント",
		"col_plan":               "Plan",
		"col_5h":                 "5h",
		"col_weekly":             "週次",
		"col_status":             "状態",
		"accounts_hint":          "チェック＝許可リスト。各カードで「共通に従う」か「独自時刻」を選べます。5h／週次は表示しません。",
		"actions":                "操作",
		"key_placeholder":        "Management Key（同一タブのみ保持・タブを閉じると消えます）",
		"save":                   "設定を保存",
		"run_now":                "今すぐ実行",
		"refresh":                "再読み込み",
		"refresh_models":         "モデル更新",
		"actions_hint":           "保存：ホストの plugins.configs を PATCH。今すぐ実行：保存済み許可リスト内のアカウントのみ ping（先に選択を保存してください）。",
		"last_run":               "直近の報告",
		"last_run_sub":           "レシート形式の要約です。",
		"last_run_receipt_label": "最新",
		"last_run_empty_title":   "まだ実行していません",
		"last_run_empty_body":    "許可リストを保存して「今すぐ実行」するか、スケジュールを待ってください。結果は data/codex-selective-ping に残ります。",
		"no_last_run":            "実行記録はまだありません",
		"run_history":            "実行履歴",
		"run_history_title":      "実行と履歴",
		"run_history_sub":        "最新レシート＋展開可能な履歴。手動（force）とスケジュールの両方を記録し、展開でアカウント別結果を見ます。",
		"col_result":             "結果",
		"col_http":               "HTTP",
		"col_detail":             "詳細",
		"need_key":               "Management Key が必要です",
		"saving":                 "保存中...",
		"starting":               "開始中...",
		"run_polling":            "実行中。完了後に自動更新します…",
		"run_already":            "すでに実行中です。完了を待ちます…",
		"quota_remain":           "残",
		"quota_used":             "使用",
		"quota_reset":            "リセット",
		"auth_index_label":       "認証ID",
		"chip_mode":              "モード",
		"chip_ok":                "成功",
		"chip_limited":           "制限",
		"chip_failed":            "失敗",
		"chip_skipped":           "スキップ",
		"status_success":         "成功",
		"status_limited":         "制限",
		"status_failed":          "失敗",
		"status_skipped":         "スキップ",
		"rhythm_sub":             "「共通に従う」アカウントはこの時刻を使います。独自設定は各カードで編集します。",
		"slot_next_inherit":      "次（継承アカウント）",
		"how_title":              "何時に発火する？",
		"how_body":               "選択中アカウントの有効時刻の和集合で待ち、その HH:MM を含むアカウントだけ ping します。",
		"how_global_off":         "全体オフ",
		"how_global_off_v":       "全スケジュール停止",
		"how_manual":             "手動",
		"how_manual_v":           "許可リスト全員",
		"accounts_times_sub":     "チェック＝許可リスト。各カードで継承／独自時刻。5h／週次は表示しません。",
		"sched_inherit":          "共通に従う",
		"sched_custom":           "独自",
		"sched_effective":        "有効：",
		"sched_not_selected":     "許可リスト外 — 保存時に無視／prune されます",
		"sched_custom_empty":     "独自時刻が空なら、保存時に共通リズムへ戻ります。",
		"next_custom_only":       "次回実行 %s（独自アカウントのみ；共通タイムライン外）",
		"sched_add":              "追加",
		"hist_expand":            "展開",
		"hist_collapse":          "折りたたみ",
		"hist_attempts":          "attempts",
		"hist_pager_prev":        "前へ",
		"hist_pager_next":        "次へ",
		"hist_pager_per_page":    "件数",
		"mode_force":             "手動実行",
		"mode_scheduled":         "スケジュール",
		"chip_unselected":        "未選択",
		"accounts_none_title":    "Codex アカウントがありません",
		"accounts_none_body":     "CPA 認証ファイルが出るとカードが並びます。先に共通リズムを設定できます。",
		"accounts_none_cta":      "アカウントを再読込",
		"lang_zh":                "繁中",
		"lang_en":                "English",
		"lang_ja":                "日本語",
	},
}
