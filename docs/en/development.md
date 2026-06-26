# Development

**English** | [Русский](../ru/development.md)

## Requirements

- Go version from `go.mod`;
- GNU Make;
- Node.js for LuCI JavaScript and Markdown checks;
- ShellCheck;
- Linux x86_64, `curl`, `zstd`, and an official OpenWrt SDK for package builds.

## Checks

```sh
make test
make race
make vet
make lint
make ci
```

`make lint` checks Go formatting, `go vet`, shell scripts, LuCI JavaScript,
Markdown, JSON, Python syntax, and GitHub Actions. `make ci` also runs regular
and race tests.

## Local daemon build

```sh
make build
make cross-build
make check-config
```

## OpenWrt package build

The generic builder downloads the exact official SDK from OpenWrt downloads,
verifies its SHA-256, and builds the daemon, LuCI, and Russian localization:

```sh
./scripts/build-openwrt.sh 24.10.7 mediatek mt7622 ./dist
./scripts/build-openwrt.sh 25.12.4 mediatek mt7622 ./dist
```

For AX6S/OpenWrt 24.10.7 there is a short wrapper:

```sh
./scripts/build-openwrt-24.10.7-ax6s.sh
```

Official SDKs run on Linux x86_64. The output contains packages, `SHA256SUMS`,
and `build-metadata.json`.

For any supported official combination the syntax is the same:

```sh
./scripts/build-openwrt.sh OPENWRT_VERSION TARGET SUBTARGET ./dist
```

`TARGET` and `SUBTARGET` come from the `release.target` field of
`ubus call system board`. For example, `mediatek/filogic` becomes:

```sh
./scripts/build-openwrt.sh 25.12.4 mediatek filogic ./dist
```

## GitHub Actions

- `ci.yml` — Go tests, race, fuzz smoke, static checks, IPK and APK smoke builds;
- `build-one.yml` — manual exact version/target/subtarget build;
- `build-openwrt.yml` — reusable SDK build;
- `release.yml` — two default `mediatek/mt7622` builds for OpenWrt 24.10.7 and
  25.12.4, manifest, checksums, and GitHub Release.

The release tag must match `PKG_VERSION`, for example `v0.2.0`. Actions are
pinned by commit SHA. Until the first successful GitHub workflow run, the
release pipeline is implemented locally but not confirmed by a public release.

`build-one.yml` is available to users through a fork and is intended for all
other versions and platforms. A custom build is not added to the main project
release automatically.

## Repository layout

```text
cmd/failsafe-dns-proxy/              daemon and CLI
internal/                            config, DNS, failover, health, status
package/failsafe-dns-proxy/          daemon package and OpenWrt integration
package/luci-app-failsafe-dns-proxy/ LuCI, RPC/ACL, and ru localization
scripts/                             build, release, installer, soak helpers
tests/                               integration and shell tests
docs/PLAN.md                         roadmap and status ledger
```

Behavior changes should include a test at the lowest useful level and an
update to [PLAN.md](../PLAN.md).

[← Documentation index](index.md)
