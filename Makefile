GO ?= go

.PHONY: test race vet build cross-build check-config

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o build/failsafe-dns-proxy ./cmd/failsafe-dns-proxy

cross-build:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -trimpath -ldflags="-s -w" -o build/failsafe-dns-proxy-linux-arm64 ./cmd/failsafe-dns-proxy

check-config:
	$(GO) run ./cmd/failsafe-dns-proxy check-config --config package/failsafe-dns-proxy/files/etc/config/failsafe-dns-proxy
