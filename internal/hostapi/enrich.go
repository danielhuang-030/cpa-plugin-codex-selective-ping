package hostapi

import (
	"context"
	"encoding/json"
)

// EnrichFromHost fills Plan/5h/weekly from AuthGetRuntime and AuthGet credential JSON.
// Selection/ping logic is unchanged; missing fields stay empty (UI shows —).
func EnrichFromHost(ctx context.Context, h Host, a AuthFile) AuthFile {
	var runtime json.RawMessage
	if raw, err := h.AuthGetRuntime(ctx, a.AuthIndex); err == nil {
		runtime = raw
	}
	var cred json.RawMessage
	if raw, err := h.AuthGet(ctx, a.AuthIndex); err == nil {
		cred = json.RawMessage(raw)
	}
	return EnrichQuota(a, runtime, cred)
}
