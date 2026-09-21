# Codex Selective Ping (CPA plugin)

[English](README.md) | [繁體中文](README.zh-Hant.md) | [日本語](README.ja.md)

Independent [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) plugin. Inspired by [`cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping), but it only pings accounts listed in `accounts`. An empty `accounts` list means nobody is pinged.

- Plugin ID: `codex-selective-ping`
- Fixed model: `gpt-5.6-luna`
- Config persistence: host `plugins.configs.codex-selective-ping`
- Management UI: Traditional Chinese / English / Japanese (default follows the CPA Management Center language)

## Install

### 1. Plugin Store (recommended)

In CPA Management Center, add this custom plugin source, then install **Codex Selective Ping**:

```text
https://raw.githubusercontent.com/danielhuang-030/cpa-plugin-codex-selective-ping/main/registry.json
```

Place the shared library under CPA’s plugin directory (often `plugins/`), or let the store unpack the release zip there:

```text
codex-selective-ping_0.1.3_linux_amd64.zip
└── codex-selective-ping.so
```

### 2. Configure CPA

```yaml
plugins:
  enabled: true
  dir: plugins
  configs:
    codex-selective-ping:
      enabled: true
      schedule_enabled: true
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

- `accounts` matches `auth_index` (exact) or email / name / account (case-insensitive).
- Empty `accounts` → scheduled and manual runs attempt 0 pings.
- Does not ping on CPA startup; waits for the next configured time.

Restart or reload CPA if needed, then open:

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

### 3. Manual install (optional)

If you are not using the store:

```bash
git clone https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping.git
cd cpa-plugin-codex-selective-ping
docker compose up -d --build
docker compose exec -T dev make build-linux
cp package/codex-selective-ping.so /path/to/cpa/plugins/
```

Then apply the YAML above and restart CPA.

## Management

API:

```text
GET  /v0/management/plugins/codex-selective-ping/status
POST /v0/management/plugins/codex-selective-ping/run
GET/PATCH /v0/management/plugins/codex-selective-ping/config
```

- `POST .../run` returns 202, or 409 if a run is already in progress.
- Manual **Run now** still works when scheduled ping is disabled (`schedule_enabled: false`); only the daily schedule is stopped.
- UI language follows CPA Management Center (`cli-proxy-language` / `Accept-Language`). Override with `?lang=zh-Hant|en|ja`. Unsupported locales fall back to Traditional Chinese. The plugin never writes CPA’s language key.
- UI theme follows CPA (`cli-proxy-theme` / `cli-proxy-color-scheme` / `theme`, then `prefers-color-scheme`). Override with `?theme=light|dark`. Applies `data-theme` on `<html>`.
- Quota columns (Plan / 5h / weekly) show host-provided values only; missing fields render as "—".

## Reference

Schedule, ping, and host ABI behaviour follow [`jiz4oh/cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping). This project is independent; the differences are the account allowlist, management UI, and config persistence.

## Build & develop (optional)

For contributors. End users can ignore this section if they install from the store.

```bash
docker compose up -d --build
docker compose exec -T dev go test ./... -count=1
docker compose exec -T dev make build-linux
# outputs package/codex-selective-ping.so
```

Locally:

```bash
make test
make build-linux
```

macOS:

```bash
CGO_ENABLED=1 go build -buildmode=c-shared -o codex-selective-ping.dylib .
```

## Releasing

CPA’s plugin store shows `registry.json` → `version` for **uninstalled** plugins (it does not query GitHub latest until installed). When cutting a release, bump **all** of:

1. `Makefile` `VERSION`
2. `registration.go` `version`
3. `registry.json` `plugins[0].version`

Then run `make verify-version` before tagging.
