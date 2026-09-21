package main

import (
	"context"
	"encoding/json"
	"fmt"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
)

// hostCallFn is set by CGO main to call into CPA host ABI.
var hostCallFn = func(method string, req []byte) ([]byte, error) {
	return nil, fmt.Errorf("host not initialized")
}

type cpaHost struct{}

func (cpaHost) AuthList(ctx context.Context) ([]hostapi.AuthFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := hostCallFn("host.auth.list", []byte(`{}`))
	if err != nil {
		return nil, err
	}
	var env struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if !env.OK {
		if env.Error != nil {
			return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		return nil, fmt.Errorf("host.auth.list failed")
	}
	var resp struct {
		Files []hostapi.AuthFile `json:"files"`
	}
	if err := json.Unmarshal(env.Result, &resp); err != nil {
		return nil, err
	}
	return resp.Files, nil
}

func (cpaHost) AuthGet(ctx context.Context, authIndex string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{"auth_index": authIndex})
	raw, err := hostCallFn("host.auth.get", payload)
	if err != nil {
		return nil, err
	}
	result, err := unwrapHost(raw)
	if err != nil {
		return nil, err
	}
	var resp struct {
		JSON json.RawMessage `json:"json"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, err
	}
	if len(resp.JSON) == 0 {
		return nil, fmt.Errorf("empty auth JSON")
	}
	return resp.JSON, nil
}

func (cpaHost) HTTPDo(ctx context.Context, req hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	if err := ctx.Err(); err != nil {
		return hostapi.HTTPResponse{}, err
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return hostapi.HTTPResponse{}, err
	}
	raw, err := hostCallFn("host.http.do", payload)
	if err != nil {
		return hostapi.HTTPResponse{}, err
	}
	result, err := unwrapHost(raw)
	if err != nil {
		return hostapi.HTTPResponse{}, err
	}
	var resp hostapi.HTTPResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return hostapi.HTTPResponse{}, err
	}
	return resp, nil
}

func (cpaHost) AuthGetRuntime(ctx context.Context, authIndex string) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{"auth_index": authIndex})
	raw, err := hostCallFn("host.auth.get_runtime", payload)
	if err != nil {
		return nil, hostapi.ErrUnsupported
	}
	result, err := unwrapHost(raw)
	if err != nil {
		return nil, hostapi.ErrUnsupported
	}
	return json.RawMessage(result), nil
}

func unwrapHost(raw []byte) (json.RawMessage, error) {
	var env struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if !env.OK {
		if env.Error != nil {
			return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		return nil, fmt.Errorf("host callback failed")
	}
	return env.Result, nil
}
