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
		"title":              "Codex Selective Ping",
		"subtitle":           "獨立 CPA 插件 · 指定帳號排程 ping",
		"overview":           "概況",
		"status":             "狀態",
		"version_model":      "版本 / 模型",
		"tz_times":           "時區 / 時段",
		"next_run":           "下次執行",
		"running_label":      "執行中",
		"yes":                "是",
		"no":                 "否",
		"enabled_on":         "已啟用",
		"enabled_off":        "停用",
		"schedule":           "排程",
		"rhythm_title":       "今天的節奏",
		"slot_next":          "即將 · 下次",
		"slot_past":          "已過",
		"enable":             "啟用",
		"timezone":           "時區",
		"add_time":           "新增時段",
		"add":                "加入",
		"schedule_hint":      "啟動時不會立刻全 ping；到點才跑。空帳號清單 = 誰都不 ping。",
		"accounts":           "帳號",
		"banner_selected":    "目前已勾選 %d / %d 個帳號。未勾選的不會被排程或「立刻執行」打到。",
		"banner_empty":       "帳號白名單為空：排程與立刻執行都不會 ping 任何人。",
		"select_all":         "全選",
		"clear":              "清除",
		"col_account":        "帳號",
		"col_plan":           "Plan",
		"col_5h":             "5h",
		"col_weekly":         "週限",
		"col_status":         "狀態",
		"accounts_hint":      "Plan 來自憑證 id_token；輸入 Management Key 後會向 CPA auth-files／api-call（wham/usage）補 Plan／5h／週限。沒有資料就顯示 —，不會自己推估。額度欄只供參考，不決定是否 ping。",
		"actions":            "動作",
		"key_placeholder":    "Management Key（僅當次輸入，不持久化）",
		"save":               "儲存設定",
		"run_now":            "立刻執行",
		"refresh":            "重新整理",
		"actions_hint":       "儲存：PATCH 宿主 plugins.configs。立刻執行：只 ping 目前已儲存白名單中的帳號（請先儲存勾選）。",
		"last_run":           "最近一次結果",
		"no_last_run":        "尚無執行紀錄",
		"col_result":         "結果",
		"col_http":           "HTTP",
		"col_detail":         "說明",
		"need_key":           "需要 Management Key",
		"saving":             "儲存中...",
		"starting":           "啟動中...",
		"quota_remain":       "剩",
		"quota_used":         "用",
		"quota_reset":        "重置",
		"auth_index_label":  "認證索引",
		"chip_mode":         "模式",
		"chip_ok":           "成功",
		"chip_limited":      "限額",
		"chip_failed":       "失敗",
		"chip_skipped":      "略過",
		"status_success":    "成功",
		"status_limited":    "限額",
		"status_failed":     "失敗",
		"status_skipped":    "略過",
		"lang_zh":            "繁中",
		"lang_en":            "English",
		"lang_ja":            "日本語",
	},
	LangEn: {
		"title":              "Codex Selective Ping",
		"subtitle":           "Standalone CPA plugin · scheduled ping for selected accounts",
		"overview":           "Overview",
		"status":             "Status",
		"version_model":      "Version / model",
		"tz_times":           "Timezone / slots",
		"next_run":           "Next run",
		"running_label":      "Running",
		"yes":                "Yes",
		"no":                 "No",
		"enabled_on":         "Enabled",
		"enabled_off":        "Disabled",
		"schedule":           "Schedule",
		"rhythm_title":       "Today's rhythm",
		"slot_next":          "Up next",
		"slot_past":          "Past",
		"enable":             "Enable",
		"timezone":           "Timezone",
		"add_time":           "Add time",
		"add":                "Add",
		"schedule_hint":      "Startup does not ping everyone; runs only at scheduled times. An empty account list means nobody is pinged.",
		"accounts":           "Accounts",
		"banner_selected":    "%d / %d accounts selected. Unchecked accounts are skipped by the schedule and Run now.",
		"banner_empty":       "Account allowlist is empty: neither the schedule nor Run now will ping anyone.",
		"select_all":         "Select all",
		"clear":              "Clear",
		"col_account":        "Account",
		"col_plan":           "Plan",
		"col_5h":             "5h",
		"col_weekly":         "Weekly",
		"col_status":         "Status",
		"accounts_hint":      "Plan comes from the credential id_token; with a Management Key the page fills Plan/5h/weekly from CPA auth-files and api-call (wham/usage). Missing values stay — and are never estimated. Quota is informational and does not decide whether to ping.",
		"actions":            "Actions",
		"key_placeholder":    "Management Key (entered for this session only, not persisted)",
		"save":               "Save settings",
		"run_now":            "Run now",
		"refresh":            "Refresh",
		"actions_hint":       "Save: PATCH host plugins.configs. Run now: pings only accounts already saved in the allowlist (save your selection first).",
		"last_run":           "Last run",
		"no_last_run":        "No runs yet",
		"col_result":         "Result",
		"col_http":           "HTTP",
		"col_detail":         "Detail",
		"need_key":           "Management Key required",
		"saving":             "Saving...",
		"starting":           "Starting...",
		"quota_remain":       "left",
		"quota_used":         "used",
		"quota_reset":        "resets",
		"auth_index_label":  "auth_index",
		"chip_mode":         "mode",
		"chip_ok":           "ok",
		"chip_limited":      "limited",
		"chip_failed":       "failed",
		"chip_skipped":      "skipped",
		"status_success":    "success",
		"status_limited":    "limited",
		"status_failed":     "failed",
		"status_skipped":    "skipped",
		"lang_zh":            "繁中",
		"lang_en":            "English",
		"lang_ja":            "日本語",
	},
	LangJa: {
		"title":              "Codex Selective Ping",
		"subtitle":           "独立 CPA プラグイン · 指定アカウントのスケジュール ping",
		"overview":           "概要",
		"status":             "状態",
		"version_model":      "バージョン / モデル",
		"tz_times":           "タイムゾーン / 時刻",
		"next_run":           "次回実行",
		"running_label":      "実行中",
		"yes":                "はい",
		"no":                 "いいえ",
		"enabled_on":         "有効",
		"enabled_off":        "無効",
		"schedule":           "スケジュール",
		"rhythm_title":       "今日のリズム",
		"slot_next":          "次",
		"slot_past":          "済",
		"enable":             "有効",
		"timezone":           "タイムゾーン",
		"add_time":           "時刻を追加",
		"add":                "追加",
		"schedule_hint":      "起動時に全員へ ping しません。予定時刻のみ実行します。アカウント一覧が空なら誰にも ping しません。",
		"accounts":           "アカウント",
		"banner_selected":    "現在 %d / %d アカウントを選択中。未選択はスケジュールと「今すぐ実行」の対象外です。",
		"banner_empty":       "アカウント許可リストが空です。スケジュールも今すぐ実行も誰にも ping しません。",
		"select_all":         "すべて選択",
		"clear":              "クリア",
		"col_account":        "アカウント",
		"col_plan":           "Plan",
		"col_5h":             "5h",
		"col_weekly":         "週次",
		"col_status":         "状態",
		"accounts_hint":      "Plan は id_token 由来。Management Key 入力後に CPA auth-files／api-call（wham/usage）から Plan／5h／週次を補完します。無い場合は —（推測しません）。クォータ欄は参考情報で、ping 可否は決めません。",
		"actions":            "操作",
		"key_placeholder":    "Management Key（この入力のみ・保存しません）",
		"save":               "設定を保存",
		"run_now":            "今すぐ実行",
		"refresh":            "再読み込み",
		"actions_hint":       "保存：ホストの plugins.configs を PATCH。今すぐ実行：保存済み許可リスト内のアカウントのみ ping（先に選択を保存してください）。",
		"last_run":           "直近の結果",
		"no_last_run":        "実行記録はまだありません",
		"col_result":         "結果",
		"col_http":           "HTTP",
		"col_detail":         "詳細",
		"need_key":           "Management Key が必要です",
		"saving":             "保存中...",
		"starting":           "開始中...",
		"quota_remain":       "残",
		"quota_used":         "使用",
		"quota_reset":        "リセット",
		"auth_index_label":  "認証ID",
		"chip_mode":         "モード",
		"chip_ok":           "成功",
		"chip_limited":      "制限",
		"chip_failed":       "失敗",
		"chip_skipped":      "スキップ",
		"status_success":    "成功",
		"status_limited":    "制限",
		"status_failed":     "失敗",
		"status_skipped":    "スキップ",
		"lang_zh":            "繁中",
		"lang_en":            "English",
		"lang_ja":            "日本語",
	},
}
