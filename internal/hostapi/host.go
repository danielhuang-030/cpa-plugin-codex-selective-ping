package hostapi

import (
	"context"
	"encoding/json"
	"errors"
)

var ErrUnsupported = errors.New("unsupported host method")

type HTTPRequest struct {
	Method  string              `json:"Method"`
	URL     string              `json:"URL"`
	Headers map[string][]string `json:"Headers,omitempty"`
	Body    []byte              `json:"Body,omitempty"`
}

type HTTPResponse struct {
	StatusCode int                 `json:"StatusCode"`
	Headers    map[string][]string `json:"Headers,omitempty"`
	Body       []byte              `json:"Body,omitempty"`
}

type Host interface {
	AuthList(ctx context.Context) ([]AuthFile, error)
	AuthGet(ctx context.Context, authIndex string) ([]byte, error)
	HTTPDo(ctx context.Context, req HTTPRequest) (HTTPResponse, error)
	// AuthGetRuntime is optional; return ErrUnsupported when the host ABI lacks it.
	AuthGetRuntime(ctx context.Context, authIndex string) (json.RawMessage, error)
}
