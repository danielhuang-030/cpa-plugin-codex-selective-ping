package main

import (
	"encoding/json"
	"fmt"

	"cpa-plugin-codex-selective-ping/internal/config"
)

// registerConfigRequest mirrors CPA rpcLifecycleRequest: config_yaml is []byte
// and therefore base64-encoded in the JSON RPC envelope.
type registerConfigRequest struct {
	ConfigYAML []byte `json:"config_yaml"`
}

func parseRegisterConfigJSON(requestBytes []byte) (config.Config, error) {
	var req registerConfigRequest
	if len(requestBytes) > 0 {
		if err := json.Unmarshal(requestBytes, &req); err != nil {
			return config.Config{}, fmt.Errorf("invalid plugin configuration envelope: %w", err)
		}
	}
	return config.Parse(string(req.ConfigYAML))
}
