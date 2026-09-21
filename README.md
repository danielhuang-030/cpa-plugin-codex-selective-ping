# Codex Selective Ping (CPA plugin)

[English](README.md) | [繁體中文](README.zh-Hant.md) | [日本語](README.ja.md)

Independent [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) plugin. Inspired by [`cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping), but it only pings accounts listed in `accounts`. An empty `accounts` list means nobody is pinged.

- Plugin ID: `codex-selective-ping`
- Fixed model: `gpt-5.6-luna`
- Config persistence: host `plugins.configs.codex-selective-ping`
- Management UI: Traditional Chinese / English / Japanese (default follows the CPA Management Center language)

## Reference

This plugin is a new, independent project. Schedule, ping, and host ABI behaviour follow [`jiz4oh/cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping). The differences are the account allowlist, management UI, and config persistence.

## Install

### Option A — CPA Plugin Store (recommended)

1. In CPA Management Center, add this custom plugin source:

```text
https://raw.githubusercontent.com/danielhuang-030/cpa-plugin-codex-selective-ping/main/registry.json
```

2. Install **Codex Selective Ping** from the store.
3. Put the shared library under CPA’s plugin directory (often `plugins/`), or let the store place the release zip there. Release layout example:

```text
codex-selective-ping_0.1.0_linux_amd64.zip
└── codex-selective-ping.so
```

4. Enable plugins and add config (see below), then restart or reload CPA if needed.
5. Open the management resource page: `/v0/resource/plugins/codex-selective-ping/status`

### Option B — Build and copy manually

```bash
git clone https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping.git
cd cpa-plugin-codex-selective-ping
docker compose up -d --build
docker compose exec -T dev make build-linux
# or: CGO_ENABLED=1 go build -buildmode=c-shared -o codex-selective-ping.so .
cp package/codex-selective-ping.so /path/to/cpa/plugins/
```

Then set `plugins.enabled: true`, point `plugins.dir` at that directory, add `plugins.configs.codex-selective-ping`, and restart CPA.

## CPA configuration

```yaml
plugins:
  enabled: true
  dir: plugins
  configs:
    codex-selective-ping:
      enabled: true
      timezone: Asia/Taipei
      times:
        - "06:00"
        - "11:00"
        - "16:00"
        - "21:00"
      accounts:
        - "user@example.com"
        - "auth_index_or_name"
```

Semantics:

- `accounts` matches `auth_index` (exact) or email / name / account (case-insensitive).
- Empty `accounts` → scheduled and manual runs attempt 0 pings.
- Does not ping on CPA startup; waits for the next configured time.

## Management

Resource page:

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

API:

```text
GET  /v0/management/plugins/codex-selective-ping/status
POST /v0/management/plugins/codex-selective-ping/run
```

Save settings from the UI via the host:

```text
GET/PATCH /v0/management/plugins/codex-selective-ping/config
```

`POST .../run` returns 202, or 409 if a run is already in progress. Manual **Run now** still works when scheduled ping is disabled (`enabled: false`); only the daily schedule is stopped.

The resource page language follows the CPA Management Center (`cli-proxy-language` / `Accept-Language`). Override with `?lang=zh-Hant|en|ja`. Unsupported locales fall back to Traditional Chinese. The plugin never writes CPA’s language key.

Quota columns (Plan / 5h / weekly) show host-provided values only; missing fields render as "—".

## Build

Prefer Docker Compose (Go 1.24 + gcc/CGO):

```bash
docker compose up -d --build
docker compose exec -T dev go test ./... -count=1
docker compose exec -T dev make build-linux
# outputs package/codex-selective-ping.so
```

Or locally:

```bash
make test
make build-linux
```

macOS:

```bash
CGO_ENABLED=1 go build -buildmode=c-shared -o codex-selective-ping.dylib .
```

Copy the shared library into CPA’s plugin directory.

## Develop

```bash
docker compose exec -T dev go test ./...
```
