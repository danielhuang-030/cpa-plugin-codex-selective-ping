# Codex Selective Ping (CPA plugin)

Brand-new independent CLIProxyAPI plugin. Like `codex-auto-ping`, but only pings accounts listed in `accounts`. Empty `accounts` means ping nobody.

- Plugin ID: `codex-selective-ping`
- Fixed model: `gpt-5.6-luna`
- Config persistence: host `plugins.configs.codex-selective-ping`
- Management UI: Traditional Chinese resource page

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

- `accounts` matches `auth_index` (exact) or email/name/account (case-insensitive).
- Empty `accounts` → scheduled and manual runs attempt 0 pings.
- Does not ping on CPA startup; waits for next configured time.

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

Save settings from the UI via host:

```text
GET/PATCH /v0/management/plugins/codex-selective-ping/config
```

`POST .../run` returns 202, or 409 if a run is already in progress.

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

Copy the shared library into CPA's plugin directory.

## Develop

```bash
docker compose exec -T dev go test ./...
```
