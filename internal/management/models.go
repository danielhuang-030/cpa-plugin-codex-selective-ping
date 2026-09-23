package management

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/modelfilter"
	"cpa-plugin-codex-selective-ping/internal/pinger"
	"cpa-plugin-codex-selective-ping/internal/selector"
)

type modelsListItem struct {
	ID string `json:"id"`
}

type modelsListResponse struct {
	Data    []modelsListItem `json:"data"`
	Warning string           `json:"warning,omitempty"`
}

func bearerFromAuthHeader(headers map[string][]string) string {
	raw := firstHeader(headers, "Authorization")
	if raw == "" {
		return ""
	}
	const p = "bearer "
	if len(raw) >= len(p) && strings.EqualFold(raw[:len(p)], p) {
		return strings.TrimSpace(raw[len(p):])
	}
	return ""
}


func firstAPIKeyFromBody(body []byte) (string, error) {
	var root struct {
		APIKeys []json.RawMessage `json:"api-keys"`
	}
	if err := json.Unmarshal(body, &root); err != nil {
		return "", err
	}
	for _, raw := range root.APIKeys {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			s = strings.TrimSpace(s)
			if s != "" {
				return s, nil
			}
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}
		for _, k := range []string{"api-key", "apiKey", "key"} {
			if v, ok := obj[k]; ok {
				if s, ok := v.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						return s, nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("no api key")
}

func firstCodexAccessToken(ctx context.Context, h hostapi.Host) (string, error) {
	if h == nil {
		return "", fmt.Errorf("nil host")
	}
	files, err := h.AuthList(ctx)
	if err != nil {
		return "", err
	}
	for _, f := range files {
		if !selector.IsCodex(f) {
			continue
		}
		raw, err := h.AuthGet(ctx, f.AuthIndex)
		if err != nil {
			continue
		}
		tok, err := pinger.AccessTokenFromAuthJSON(raw)
		if err != nil || tok == "" {
			continue
		}
		return tok, nil
	}
	return "", fmt.Errorf("no codex access token")
}

func fetchFilteredModels(ctx context.Context, h hostapi.Host, origin, managementKey, cfgModel string) modelsListResponse {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	cfgModel = strings.TrimSpace(cfgModel)

	var lastWarn string
	if origin == "" {
		lastWarn = "models: empty origin (missing Host/X-Csp-Origin)"
	}
	try := func(token, label string) ([]modelsListItem, bool) {
		if token == "" || origin == "" || h == nil {
			return nil, false
		}
		resp, err := h.HTTPDo(ctx, hostapi.HTTPRequest{
			Method: "GET",
			URL:    origin + "/v1/models",
			Headers: map[string][]string{
				"Authorization": {"Bearer " + token},
				"Accept":        {"application/json"},
			},
		})
		if err != nil {
			lastWarn = fmt.Sprintf("models: %s: %v", label, err)
			return nil, false
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastWarn = fmt.Sprintf("models: %s: HTTP %d", label, resp.StatusCode)
			return nil, false
		}
		items, err := parseAndFilterModelsJSON(resp.Body)
		if err != nil {
			lastWarn = fmt.Sprintf("models: %s: %v", label, err)
			return nil, false
		}
		if len(items) == 0 {
			lastWarn = fmt.Sprintf("models: %s: empty after filter", label)
			return nil, false
		}
		return items, true
	}


	if managementKey != "" && origin != "" && h != nil {
		resp, err := h.HTTPDo(ctx, hostapi.HTTPRequest{
			Method: "GET",
			URL:    origin + "/v0/management/api-keys",
			Headers: map[string][]string{
				"Authorization": {"Bearer " + managementKey},
				"Accept":        {"application/json"},
			},
		})
		if err != nil {
			lastWarn = fmt.Sprintf("models: api-keys: %v", err)
		} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastWarn = fmt.Sprintf("models: api-keys: HTTP %d", resp.StatusCode)
		} else if apiKey, err := firstAPIKeyFromBody(resp.Body); err != nil {
			lastWarn = fmt.Sprintf("models: api-keys: %v", err)
		} else if items, ok := try(apiKey, "api-key"); ok {
			return modelsListResponse{Data: items}
		}
	}

	if tok, err := firstCodexAccessToken(ctx, h); err == nil {
		if items, ok := try(tok, "codex"); ok {
			return modelsListResponse{Data: items}
		}
	} else if lastWarn == "" {
		lastWarn = "models: no codex token: " + err.Error()
	}

	return modelsListResponse{Data: fallbackModelItems(cfgModel), Warning: fallbackWarning(lastWarn)}
}


func fallbackWarning(last string) string {
	if last == "" {
		return "models: upstream failed, using fallback"
	}
	if strings.Contains(last, "fallback") {
		return last
	}
	return last + "; using fallback"
}

func fallbackModelItems(cfgModel string) []modelsListItem {
	seen := map[string]bool{}
	out := make([]modelsListItem, 0, 2)
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, modelsListItem{ID: id})
	}
	add(cfgModel)
	add(pinger.ModelName)
	return out
}

func parseAndFilterModelsJSON(body []byte) ([]modelsListItem, error) {
	var root struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	out := make([]modelsListItem, 0, len(root.Data))
	seen := map[string]bool{}
	for _, m := range root.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" || seen[id] {
			continue
		}
		if !modelfilter.Keep(id, m.OwnedBy) {
			continue
		}
		seen[id] = true
		out = append(out, modelsListItem{ID: id})
	}
	return out, nil
}
