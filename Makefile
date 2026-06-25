GO ?= go
NPM ?= npx

.PHONY: test race vet fmt shellcheck markdownlint actionlint static lint ci build cross-build check-config

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

fmt:
	test -z "$$(gofmt -l cmd internal)"

shellcheck:
	shellcheck -s bash -e SC1091,SC2034,SC2153,SC2154 scripts/*.sh scripts/lib/*.sh \
		package/failsafe-dns-proxy/files/etc/init.d/failsafe-dns-proxy \
		package/failsafe-dns-proxy/files/usr/sbin/failsafe-dns-proxy-dnsmasq \
		package/failsafe-dns-proxy/files/usr/sbin/failsafe-dns-proxy-soak \
		package/luci-app-failsafe-dns-proxy/root/usr/libexec/rpcd/luci.failsafe-dns-proxy

markdownlint:
	$(NPM) --yes markdownlint-cli2@0.22.1

actionlint:
	$(GO) run github.com/rhysd/actionlint/cmd/actionlint@v1.7.9

static: fmt vet shellcheck markdownlint actionlint
	node --check package/luci-app-failsafe-dns-proxy/htdocs/luci-static/resources/view/failsafe-dns-proxy/overview.js
	python3 -m compileall -q scripts
	python3 -m unittest discover -s tests -p 'test_*.py'
	python3 -m json.tool build/supported-openwrt.json >/dev/null
	python3 -m json.tool build/release-targets.json >/dev/null
	tests/shell/run.sh

lint: static

ci: lint test race

build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o build/failsafe-dns-proxy ./cmd/failsafe-dns-proxy

cross-build:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -trimpath -ldflags="-s -w" -o build/failsafe-dns-proxy-linux-arm64 ./cmd/failsafe-dns-proxy

check-config:
	$(GO) run ./cmd/failsafe-dns-proxy check-config --config package/failsafe-dns-proxy/files/etc/config/failsafe-dns-proxy
