package modelfilter

import "testing"

func TestKeepCodexOpenAIChatGPT(t *testing.T) {
	cases := []struct {
		id, owned string
		want      bool
	}{
		{"gpt-6-luna", "openai", true},
		{"codex-mini", "", true},
		{"chatgpt-4o", "openai", true},
		{"claude-sonnet", "anthropic", false},
		{"gemini-pro", "google", false},
		{"GPT-4o", "OpenAI", true},
	}
	for _, tc := range cases {
		if got := Keep(tc.id, tc.owned); got != tc.want {
			t.Fatalf("%s/%s: got %v want %v", tc.id, tc.owned, got, tc.want)
		}
	}
}
