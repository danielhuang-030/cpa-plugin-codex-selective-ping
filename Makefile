PLUGIN_ID=codex-selective-ping
VERSION=0.1.0

.PHONY: test build-linux clean

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
