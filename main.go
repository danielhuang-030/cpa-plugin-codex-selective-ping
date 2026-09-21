//go:build cgo

package main

/*
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef struct cliproxy_buffer { uint8_t* ptr; size_t len; } cliproxy_buffer;
typedef int (*cliproxy_host_call_fn)(void* host_ctx, const char* method,
    const uint8_t* request, size_t request_len, cliproxy_buffer* response);
typedef void (*cliproxy_host_free_buffer_fn)(void* ptr, size_t len);
typedef struct cliproxy_host_api {
    uint32_t abi_version; void* host_ctx; cliproxy_host_call_fn call; cliproxy_host_free_buffer_fn free_buffer;
} cliproxy_host_api;
typedef int (*cliproxy_plugin_call_fn)(char* method, uint8_t* request, size_t request_len, cliproxy_buffer* response);
typedef void (*cliproxy_plugin_free_buffer_fn)(void* ptr, size_t len);
typedef void (*cliproxy_plugin_shutdown_fn)(void);
typedef struct cliproxy_plugin_api {
    uint32_t abi_version; cliproxy_plugin_call_fn call; cliproxy_plugin_free_buffer_fn free_buffer; cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;
#ifdef _WIN32
#define CPA_PLUGIN_EXPORT __declspec(dllexport)
#else
#define CPA_PLUGIN_EXPORT
#endif
extern CPA_PLUGIN_EXPORT int selectivePingPluginCall(char* method, uint8_t* request, size_t request_len, cliproxy_buffer* response);
extern CPA_PLUGIN_EXPORT void selectivePingPluginFreeBuffer(void* ptr, size_t len);
extern CPA_PLUGIN_EXPORT void selectivePingPluginShutdown(void);
static const cliproxy_host_api* stored_host;
static inline void store_host_api(const cliproxy_host_api* host) { stored_host = host; }
static inline void set_plugin_api(cliproxy_plugin_api* plugin) {
    plugin->abi_version = 1; plugin->call = selectivePingPluginCall; plugin->free_buffer = selectivePingPluginFreeBuffer; plugin->shutdown = selectivePingPluginShutdown;
}
static inline int call_host_api(const char* method, const uint8_t* request, size_t request_len, cliproxy_buffer* response) {
    if (stored_host == NULL || stored_host->call == NULL) return 1;
    return stored_host->call(stored_host->host_ctx, method, request, request_len, response);
}
static inline void free_host_buffer(void* ptr, size_t len) {
    if (stored_host != NULL && stored_host->free_buffer != NULL && ptr != NULL) stored_host->free_buffer(ptr, len);
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/management"
	"cpa-plugin-codex-selective-ping/internal/plugin"
)

var app = plugin.New(cpaHost{}, version)
var mgmt = &management.Handler{Plugin: app}

func main() {}

func init() {
	hostCallFn = func(method string, req []byte) ([]byte, error) {
		cm := C.CString(method)
		defer C.free(unsafe.Pointer(cm))
		var reqPtr *C.uint8_t
		var cPayload unsafe.Pointer
		if len(req) > 0 {
			cPayload = C.CBytes(req)
			if cPayload == nil {
				return nil, fmt.Errorf("allocation failure")
			}
			defer C.free(cPayload)
			reqPtr = (*C.uint8_t)(cPayload)
		}
		var response C.cliproxy_buffer
		code := C.call_host_api(cm, reqPtr, C.size_t(len(req)), &response)
		data := copyHostResponse(response)
		if response.ptr != nil {
			C.free_host_buffer(unsafe.Pointer(response.ptr), response.len)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("host callback %s empty response code=%d", method, int(code))
		}
		if code != 0 {
			return data, fmt.Errorf("host callback code=%d", int(code))
		}
		return data, nil
	}
}

//export cliproxy_plugin_init
func cliproxy_plugin_init(host *C.cliproxy_host_api, pluginAPI *C.cliproxy_plugin_api) C.int {
	if host == nil || pluginAPI == nil {
		return -1
	}
	C.store_host_api(host)
	C.set_plugin_api(pluginAPI)
	return 0
}

//export selectivePingPluginCall
func selectivePingPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response == nil {
		return -1
	}
	name := ""
	if method != nil {
		name = C.GoString(method)
	}
	requestBytes, ok := copyRequestBytes(request, requestLen)
	if !ok {
		return writeJSON(response, failEnvelope("invalid_request", "invalid request length"))
	}
	switch name {
	case "plugin.register", "plugin.reconfigure":
		cfg, err := parseRegisterConfigJSON(requestBytes)
		if err != nil {
			return writeJSON(response, failEnvelope("invalid_config", err.Error()))
		}
		app.ApplyConfig(cfg)
		return writeJSON(response, okEnvelope(registrationResult()))
	case "management.register":
		return writeJSON(response, okEnvelope(managementRegistrationResult()))
	case "management.handle":
		var req management.Request
		if json.Unmarshal(requestBytes, &req) != nil {
			return writeJSON(response, failEnvelope("invalid_request", "invalid management request"))
		}
		resp := mgmt.Handle(req)
		return writeJSON(response, okEnvelope(resp))
	case "plugin.shutdown":
		app.Shutdown()
		return writeJSON(response, okEnvelope(map[string]any{"status": "stopped"}))
	default:
		return writeJSON(response, failEnvelope("unsupported_method", "unsupported plugin method"))
	}
}

//export selectivePingPluginFreeBuffer
func selectivePingPluginFreeBuffer(ptr unsafe.Pointer, length C.size_t) {
	_ = length
	C.free(ptr)
}

//export selectivePingPluginShutdown
func selectivePingPluginShutdown() { app.Shutdown() }

func registrationResult() map[string]any {
	return registrationMeta(config.DefaultConfig())
}

func managementRegistrationResult() map[string]any {
	return map[string]any{
		"routes": []map[string]string{
			{"Method": "GET", "Path": "/plugins/codex-selective-ping/status", "Description": "JSON status"},
			{"Method": "POST", "Path": "/plugins/codex-selective-ping/run", "Description": "Run now for selected accounts"},
		},
		"resources": []map[string]string{
			{"Path": "/status", "Menu": "Codex Selective Ping", "Description": "選擇帳號、排程、儲存與執行"},
		},
	}
}

func copyRequestBytes(request *C.uint8_t, requestLen C.size_t) ([]byte, bool) {
	length := int(requestLen)
	if length < 0 || C.size_t(length) != requestLen {
		return nil, false
	}
	if length == 0 {
		return nil, true
	}
	if request == nil {
		return nil, false
	}
	s := unsafe.Slice((*byte)(unsafe.Pointer(request)), length)
	return append([]byte(nil), s...), true
}

func copyHostResponse(r C.cliproxy_buffer) []byte {
	if r.ptr == nil || r.len == 0 {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(r.ptr), C.int(r.len))
}

func writeJSON(response *C.cliproxy_buffer, data []byte) C.int {
	if len(data) == 0 {
		response.ptr = nil
		response.len = 0
		return 0
	}
	ptr := C.malloc(C.size_t(len(data)))
	if ptr == nil {
		return -1
	}
	C.memcpy(ptr, unsafe.Pointer(&data[0]), C.size_t(len(data)))
	response.ptr = (*C.uint8_t)(ptr)
	response.len = C.size_t(len(data))
	return 0
}
