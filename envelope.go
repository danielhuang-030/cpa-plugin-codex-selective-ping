package main

import "encoding/json"

func okEnvelope(result any) []byte {
	b, _ := json.Marshal(map[string]any{"ok": true, "result": result})
	return b
}

func failEnvelope(code, message string) []byte {
	b, _ := json.Marshal(map[string]any{"ok": false, "error": map[string]any{"code": code, "message": message, "retryable": false}})
	return b
}
