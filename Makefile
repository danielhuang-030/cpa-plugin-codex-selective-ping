PLUGIN_ID=codex-selective-ping
VERSION=0.1.8

.PHONY: test build-linux clean verify-product verify-version

test:
	CGO_ENABLED=0 go test ./... -count=1

build-linux:
	mkdir -p package dist
	CGO_ENABLED=1 go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o package/$(PLUGIN_ID).so .
	rm -f package/$(PLUGIN_ID).h
	@test -f package/$(PLUGIN_ID).so
	@ls -la package/$(PLUGIN_ID).so
	@readelf -h package/$(PLUGIN_ID).so | head -n 5

clean:
	rm -rf package dist *.so *.h

verify-product:
	./scripts/verify-version.sh
	test ! -d cpa-plugin-codex-auto-ping
	grep -q 'module cpa-plugin-codex-selective-ping' go.mod
	grep -q '"id": "codex-selective-ping"' registry.json
	@echo "product cleanup checks OK (no nested auto-ping; selective-ping module/registry)"

# CPA plugin store shows registry.json version for uninstalled plugins
# (host skips GitHub latest until installed). Keep registry/registration in sync.
verify-version:
	./scripts/verify-version.sh
