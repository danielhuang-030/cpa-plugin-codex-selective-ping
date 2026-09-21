# Codex Selective Ping

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/danielhuang-030/cpa-plugin-codex-selective-ping)](https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping/releases)

[English](README.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md)

A [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) (CPA) plugin that sends a small real Codex request only to accounts you list. Inspired by [`cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping); an empty `accounts` list means nobody is pinged.

## Features

- **Allowlist only** — pings `accounts`; empty list → 0 attempts
- **Fixed model** — `gpt-5.6-luna` (not configurable)
- **Daily schedule** — IANA timezone + `HH:MM` times; does not ping on CPA startup
- **Management UI** — Traditional Chinese / English / Japanese (follows CPA Management Center; override with `?lang=` / `?theme=`)
- **Persisted last run** — `{CPA root}/data/codex-selective-ping/run_history.json` (never under `auth-dir` / `auths/`)
- **Plugin ID** — `codex-selective-ping` · config key `plugins.configs.codex-selective-ping`

## Requirements

- [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) with dynamic plugins enabled
- Linux `amd64` for store / prebuilt `.so` installs (see [Development](#development) for other builds)

## Installation

### 1. Add the Plugin Store source

Edit CPA’s main config (usually `config.yaml` next to the binary, or the path your deployment mounts). Under top-level `plugins:`, append this URL to `store-sources`. The official registry stays included; this only adds a custom source.

```yaml
plugins:
  enabled: true
  dir: plugins
  store-sources:
    - "https://raw.githubusercontent.com/danielhuang-030/cpa-plugin-codex-selective-ping/main/registry.json"
```

Notes:

| Do | Don’t |
| --- | --- |
| Put `store-sources` next to `enabled` / `dir` / `configs` | Nest it under `plugins.configs` |
| Use the same `config.yaml` as `port`, `auth-dir`, … | Use a separate “plugins-only” file |
| Append the URL if `store-sources` already exists | Replace the whole list unless you mean to drop other sources |

Save and **restart CPA**. In Management Center → Plugin Store, install **Codex Selective Ping**.

The store unpacks the GitHub Release zip into CPA’s plugin directory (often `plugins/`):

```text
codex-selective-ping_0.1.6_linux_amd64.zip
└── codex-selective-ping.so
```

Each release must include an asset named exactly `checksums.txt` (CPA looks up that name).

### 2. Configure the plugin

In the same `config.yaml`, under `plugins.configs`, add a key named exactly `codex-selective-ping`:

```yaml
plugins:
  enabled: true
  dir: plugins
  store-sources:
    - "https://raw.githubusercontent.com/danielhuang-030/cpa-plugin-codex-selective-ping/main/registry.json"
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
      history_limit: 60
      accounts:
        - "user@example.com"
        - "auth_index_or_name"
```

Restart or reload CPA if needed, then open the status page:

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

### 3. Manual install (optional)

Skip the store and build the shared library yourself:

```bash
git clone https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping.git
cd cpa-plugin-codex-selective-ping
docker compose up -d --build
docker compose exec -T dev make build-linux
cp package/codex-selective-ping.so /path/to/cpa/plugins/
```

Then apply the `plugins.configs.codex-selective-ping` block above and restart CPA.

## Configuration reference

| Key | Type | Description |
| --- | --- | --- |
| `enabled` | bool | Host plugin instance on/off (CPA lifecycle) |
| `schedule_enabled` | bool | Daily schedule on/off; **Run now** still works when `false` |
| `timezone` | string | IANA timezone (e.g. `Asia/Taipei`) |
| `times` | string[] | One or more `HH:MM` values |
| `accounts` | string[] | Allowlist: `auth_index` (exact) or email / name / account (case-insensitive). Empty → 0 pings |
| `data_dir` | string | Optional. Directory for `run_history.json` (relative → CPA cwd) |
| `history_limit` | int | Max stored runs (default **60**; ≤0 treated as 60) |
| `state_path` | string | Optional. Full path to the last-run file (wins over `data_dir`) |

Run history is stored newest-first in `run_history.json` (default under `{CPA root}/data/codex-selective-ping/`). A legacy single-object `last_run.json` in the same directory is migrated once and then deleted.

Default history path: `{CPA root}/data/codex-selective-ping/run_history.json` (parent of `plugins/`).

## Management

### Status page

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

- Language: CPA Management Center (`cli-proxy-language` / `Accept-Language`); override `?lang=zh-Hant|en|ja` (unsupported → Traditional Chinese). This plugin never writes CPA’s language key.
- Theme: CPA (`cli-proxy-theme` / `cli-proxy-color-scheme` / `theme`, then `prefers-color-scheme`); override `?theme=light|dark` (`data-theme` on `<html>`).
- Quota columns (Plan / 5h / weekly): Plan from the Codex id_token (`chatgpt_plan_type`). With a Management Key, live 5h/weekly come from CPA `auth-files` + `api-call` (same `wham/usage` path as admin). Missing values show "—" (never estimated). Quota is informational and does not change who gets pinged.

### Management API

```text
GET        /v0/management/plugins/codex-selective-ping/status
POST       /v0/management/plugins/codex-selective-ping/run
GET/PATCH  /v0/management/plugins/codex-selective-ping/config
```

`POST .../run` returns **202**, or **409** if a run is already in progress.

## Development

For contributors. End users installing from the store can skip this section.

```bash
docker compose up -d --build
docker compose exec -T dev go test ./... -count=1
docker compose exec -T dev make build-linux
# → package/codex-selective-ping.so
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

CPA’s Plugin Store shows `registry.json` → `version` for **uninstalled** plugins (GitHub “latest” is queried only after install). When cutting a release, bump all of:

1. `Makefile` `VERSION`
2. `registration.go` `version`
3. `registry.json` `plugins[0].version`

Run `make verify-version` before tagging. Each GitHub Release must include:

- `codex-selective-ping_<version>_<goos>_<goarch>.zip`
- `checksums.txt` (exact filename; a lone `checksums-<version>.txt` is not enough)

## Related projects

- [`jiz4oh/cpa-plugin-codex-auto-ping`](https://github.com/jiz4oh/cpa-plugin-codex-auto-ping) — schedule / ping / host ABI reference; this project adds the allowlist, management UI, and config persistence
- [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) — host
- [CLIProxyAPI Plugins Store](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store) — official registry format

## License

[MIT](LICENSE)
