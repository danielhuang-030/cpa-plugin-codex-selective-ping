# cpa-plugin-codex-selective-ping

CLIProxyAPI plugin that pings only selected Codex accounts on a schedule, with a Traditional Chinese management UI.

## Dev environment (Docker Compose)

```bash
docker compose up -d --build
docker compose exec dev go test ./... -count=1
docker compose exec dev make build-linux   # after Makefile exists
```

Host Go is optional; prefer the `dev` service (Go 1.24 + gcc/CGO).

## Config (host `plugins.configs`)

See `docs/superpowers/specs/2026-09-21-codex-selective-ping-design.md`.
