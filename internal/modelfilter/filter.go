package modelfilter

import "strings"

// Keep reports whether a model should be retained based on id or ownedBy
// matching OpenAI/Codex-related heuristics (case-insensitive).
func Keep(id, ownedBy string) bool {
	idLower := strings.ToLower(id)
	ownedLower := strings.ToLower(ownedBy)
	needles := []string{"codex", "openai", "chatgpt", "gpt-"}
	for _, n := range needles {
		if strings.Contains(idLower, n) || strings.Contains(ownedLower, n) {
			return true
		}
	}
	return false
}
