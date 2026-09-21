package management

import (
	"strings"
	"testing"
	"time"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/plugin"
	"cpa-plugin-codex-selective-ping/internal/runstate"
)

func TestNormalizeLang(t *testing.T) {
	cases := []struct {
		in   string
		want Lang
	}{
		{"zh-TW", LangZhHant},
		{"zh-Hant", LangZhHant},
		{"zh-HK", LangZhHant},
		{"zh-MO", LangZhHant},
		{"zh-tw", LangZhHant},
		{"zh-hant", LangZhHant},
		{"en", LangEn},
		{"en-US", LangEn},
		{"en-GB", LangEn},
		{"ja", LangJa},
		{"ja-JP", LangJa},
		{"zh-CN", LangZhHant},
		{"ru", LangZhHant},
		{"fr", LangZhHant},
		{"", LangZhHant},
		{`{"state":{"language":"zh-TW"},"version":0}`, LangZhHant},
		{`{"state":{"language":"en"},"version":0}`, LangEn},
		{`{"state":{"language":"ja"},"version":0}`, LangJa},
		{`{"state":{"language":"zh-CN"}}`, LangZhHant},
		{`{"state":{"language":"ru"}}`, LangZhHant},
	}
	for _, tc := range cases {
		if got := NormalizeLang(tc.in); got != tc.want {
			t.Fatalf("NormalizeLang(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestResolveLang(t *testing.T) {
	if got := ResolveLang("en", "ja"); got != LangEn {
		t.Fatalf("query must win: got %q", got)
	}
	if got := ResolveLang("ja-JP", "en"); got != LangJa {
		t.Fatalf("query normalize: got %q", got)
	}
	if got := ResolveLang("", "en-US,en;q=0.9"); got != LangEn {
		t.Fatalf("Accept-Language en: got %q", got)
	}
	if got := ResolveLang("", "ja,en;q=0.8"); got != LangJa {
		t.Fatalf("Accept-Language ja: got %q", got)
	}
	if got := ResolveLang("", "zh-TW,zh;q=0.9"); got != LangZhHant {
		t.Fatalf("Accept-Language zh-TW: got %q", got)
	}
	if got := ResolveLang("", "ru,en;q=0.5"); got != LangZhHant {
		t.Fatalf("ru falls back via Normalize to zh-Hant when sole? got %q — want first tag mapped", got)
	}
	// ru maps to zh-Hant per product rules
	if got := ResolveLang("", ""); got != LangZhHant {
		t.Fatalf("fallback: got %q", got)
	}
	if got := ResolveLang("zh-CN", "en"); got != LangZhHant {
		t.Fatalf("explicit zh-CN query maps to zh-Hant: got %q", got)
	}
}

func TestRenderStatusPageEnglish(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
	}, LangEn)
	for _, want := range []string{
		`lang="en"`, "Today&#39;s rhythm", "Who to ping", "Operating principles",
		"Save settings", "Run now", "Enable", "Timezone",
		"English", "繁中", "日本語",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("en page missing %q", want)
		}
	}
	if strings.Contains(html, "今天的節奏") {
		t.Fatal("en page should not show 繁中 rhythm title")
	}
}

func TestRenderStatusPageJapanese(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
	}, LangJa)
	for _, want := range []string{
		`lang="ja"`, "今日のリズム", "対象アカウント", "操作の原則",
		"設定を保存", "今すぐ実行", "有効", "タイムゾーン",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("ja page missing %q", want)
		}
	}
}

func TestRenderStatusPageSwitcherAndCPABootstrap(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
	}, LangZhHant)
	for _, want := range []string{
		"繁中", "English", "日本語",
		"cli-proxy-language",
		"localStorage.getItem",
		"?lang=",
		`lang="zh-Hant"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing bootstrap/switcher piece %q", want)
		}
	}
	if strings.Contains(html, "localStorage.setItem") {
		t.Fatal("must not write CPA or Management Key into localStorage")
	}
}

func TestHandlerHTMLRespectsLangQuery(t *testing.T) {
	p := testPlugin()
	defer p.Shutdown()
	h := &Handler{Plugin: p}

	en := h.Handle(Request{
		Method: "GET",
		Path:   "/v0/resource/plugins/codex-selective-ping/status",
		Query:  map[string][]string{"lang": {"en"}},
	})
	if en.StatusCode != 200 {
		t.Fatalf("status %d", en.StatusCode)
	}
	body := string(en.Body)
	if !strings.Contains(body, `lang="en"`) || !strings.Contains(body, "Overview") {
		t.Fatalf("expected English HTML, got snippet: %s", truncate(body, 200))
	}

	ja := h.Handle(Request{
		Method:  "GET",
		Path:    "/v0/resource/plugins/codex-selective-ping/status",
		Headers: map[string][]string{"Accept-Language": {"ja-JP,ja;q=0.9"}},
	})
	body = string(ja.Body)
	if !strings.Contains(body, `lang="ja"`) {
		t.Fatalf("Accept-Language ja should render ja html lang; got: %s", truncate(body, 120))
	}

	def := h.Handle(Request{
		Method: "GET",
		Path:   "/v0/resource/plugins/codex-selective-ping/status",
	})
	body = string(def.Body)
	if !strings.Contains(body, `lang="zh-Hant"`) || !strings.Contains(body, "概況") {
		t.Fatalf("default should be zh-Hant; got: %s", truncate(body, 120))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func testPlugin() *plugin.Plugin {
	p := plugin.New(&mh{files: nil}, "0.1.0")
	p.ApplyConfig(config.Config{Enabled: true, Timezone: "Asia/Taipei", Times: []string{"21:00"}, Accounts: []string{}})
	return p
}

func TestCatalogParity(t *testing.T) {
	zh := catalogs[LangZhHant]
	for _, lang := range []Lang{LangEn, LangJa} {
		m := catalogs[lang]
		if len(m) != len(zh) {
			t.Fatalf("%s has %d keys, zh-Hant has %d", lang, len(m), len(zh))
		}
		for k := range zh {
			if _, ok := m[k]; !ok {
				t.Fatalf("%s missing key %q", lang, k)
			}
		}
	}
}

func TestLastRunChipsTranslated(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
		Accounts: []runstate.AccountView{{AuthIndex: "idx-1", Name: "alice", Email: "a@x.com"}},
		LastRun: &runstate.Summary{
			At: time.Date(2026, 9, 21, 21, 0, 0, 0, time.UTC),
			Mode: "manual", Succeeded: 1,
			Accounts: []runstate.AccountResult{{Name: "alice", Status: "success", HTTPStatus: 200}},
		},
	}, LangJa)
	if strings.Contains(html, "mode:") || strings.Contains(html, "auth_index:") {
		t.Fatal("ja page must not keep English chip/account labels")
	}
	for _, want := range []string{"モード", "認証ID", "成功"} {
		if !strings.Contains(html, want) {
			t.Fatalf("ja last-run missing %q", want)
		}
	}
}
