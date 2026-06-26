# Failsafe DNS Proxy for OpenWrt

**English** | [Русский](docs/README.ru.md)

[![CI](https://github.com/LedyBacer/owrt-failsafe-dns-proxy-go/actions/workflows/ci.yml/badge.svg)](https://github.com/LedyBacer/owrt-failsafe-dns-proxy-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/daemon-MIT-blue.svg)](LICENSE)

**Automatic DNS upstream failover for OpenWrt when your preferred resolver
stops working.**

On a home router you often want a primary DNS path — for example, local
`https-dns-proxy` for encrypted DNS — and a plain fallback such as ISP or
public DNS. dnsmasq can point at only one upstream address at a time and does
not track resolver health. If the primary service hangs, crashes, or loses
connectivity, LAN DNS stalls until you change the configuration manually.

Failsafe DNS Proxy sits between dnsmasq and your upstream resolvers. dnsmasq
keeps serving clients, cache, and local hostnames; the daemon is the single
upstream for dnsmasq and handles the rest:

- sends traffic to upstreams in strict priority order;
- skips a failed upstream quickly instead of waiting on it for every query;
- falls back to the next configured resolver within bounded timeouts;
- probes failed upstreams in the background and returns to the preferred one
  only after confirmed recovery.

Typical use case: **primary** `127.0.0.1:5054` (`https-dns-proxy`) →
**fallback** `77.88.8.8:53`. The daemon is not a replacement for dnsmasq, not
an ad blocker or DNS cache, and not a DoH/DoT client — only a small failover
layer for upstream selection.

```text
LAN clients
    |
    v
dnsmasq
    |
    v
failsafe-dns-proxy 127.0.0.1:5359
    |-- priority 10 -> local https-dns-proxy 127.0.0.1:5054
    `-- priority 20 -> fallback DNS 77.88.8.8:53
```

Full documentation: [English](docs/en/index.md) | [Русский](docs/ru/index.md) —
start with [Overview](docs/en/overview.md).

## Installation

### From GitHub Release

After a release is published, download `install.sh` to the router and run it
with the manifest and base URL from the same release:

```sh
RELEASE=v0.2.1
BASE="https://github.com/LedyBacer/owrt-failsafe-dns-proxy-go/releases/download/$RELEASE"

uclient-fetch -O /tmp/install-failsafe-dns-proxy.sh "$BASE/install.sh"
chmod +x /tmp/install-failsafe-dns-proxy.sh

/tmp/install-failsafe-dns-proxy.sh \
  --manifest "$BASE/manifest.json" \
  --base-url "$BASE"
```

By default the installer installs the daemon, LuCI, and Russian localization.
Variants:

```sh
# Without Russian localization
/tmp/install-failsafe-dns-proxy.sh \
  --manifest "$BASE/manifest.json" \
  --base-url "$BASE" \
  --no-russian

# Daemon only
/tmp/install-failsafe-dns-proxy.sh \
  --manifest "$BASE/manifest.json" \
  --base-url "$BASE" \
  --daemon-only
```

The installer does not change dnsmasq unless you pass `--configure-dnsmasq`.
For a first install, configure and test the proxy before enabling dnsmasq
integration.

### From locally built packages

Copy `manifest.json`, the matching packages, and the installer to the router:

```sh
./install.sh \
  --manifest ./manifest.json \
  --source-dir .
```

Or install packages manually:

```sh
# OpenWrt 24.10
opkg install ./failsafe-dns-proxy_*.ipk
opkg install ./luci-app-failsafe-dns-proxy_*.ipk
opkg install ./luci-i18n-failsafe-dns-proxy-ru_*.ipk

# OpenWrt 25.12
apk --allow-untrusted add ./failsafe-dns-proxy-*.apk
apk --allow-untrusted add ./luci-app-failsafe-dns-proxy-*.apk
apk --allow-untrusted add ./luci-i18n-failsafe-dns-proxy-ru-*.apk
```

Package installation does not enable the service or change dnsmasq.

After installation, follow [Quick start](docs/en/quick-start.md).

## Build for your platform

Official releases target a small set of OpenWrt versions and platforms. Do not
install a package built for a similar router just because the CPU architecture
matches. The installer requires an exact match of OpenWrt version, target,
subtarget, and package architecture.

Check your router parameters:

```sh
ubus call system board
```

You need fields such as:

```json
{
  "release": {
    "version": "25.12.4",
    "target": "mediatek/filogic"
  }
}
```

In this example the build parameters are:

```text
openwrt_version: 25.12.4
target: mediatek
subtarget: filogic
```

Package architecture:

```sh
# OpenWrt 24.10
opkg print-architecture

# OpenWrt 25.12
apk --print-arch
```

### Automatic build via GitHub fork

1. Sign in to GitHub and open the project repository.
2. Click **Fork → Create fork**.
3. In your fork, open the **Actions** tab.
4. If GitHub shows a warning, click
   **I understand my workflows, go ahead and enable them**.
5. Select **Build one OpenWrt target** on the left.
6. Click **Run workflow**.
7. Enter exact values:
   - `openwrt_version`, for example `25.12.4`;
   - `target`, for example `mediatek`;
   - `subtarget`, for example `filogic`;
   - leave `publish_artifact` enabled.
8. Click the green **Run workflow** button and wait for the job to finish.
9. Open the completed run and download the artifact
   `openwrt-VERSION-TARGET-SUBTARGET` at the bottom of the page.

The workflow automatically:

- finds the official SDK for the requested combination;
- downloads official `sha256sums` and verifies the SDK;
- selects IPK or APK based on the OpenWrt version;
- builds the daemon, LuCI, and Russian localization;
- smoke-tests package contents;
- adds `SHA256SUMS` and `build-metadata.json`;
- fails if no official SDK or supported package format exists.

Supported official series:

- OpenWrt `24.10.x` — IPK and `opkg`;
- OpenWrt `25.12.x` — APK and `apk`.

Arbitrary older versions, development snapshots, OpenWrt forks, and future
series are not supported automatically.

### Install a custom build

Unpack the downloaded GitHub artifact on your computer and verify checksums:

```sh
sha256sum -c SHA256SUMS
```

Copy the packages to the router:

```sh
scp failsafe-dns-proxy* luci-app-failsafe-dns-proxy* \
  luci-i18n-failsafe-dns-proxy-ru* root@192.168.1.1:/tmp/
```

| Package | Required | Purpose |
| --- | --- | --- |
| `failsafe-dns-proxy` | yes | daemon, CLI, UCI, procd, dnsmasq helpers |
| `luci-app-failsafe-dns-proxy` | no | LuCI configuration and monitoring |
| `luci-i18n-failsafe-dns-proxy-ru` | no | Russian LuCI translation |

Minimal install:

```sh
# OpenWrt 24.10
opkg install /tmp/failsafe-dns-proxy_*.ipk

# OpenWrt 25.12
apk --allow-untrusted add /tmp/failsafe-dns-proxy-*.apk
```

Full install:

```sh
# OpenWrt 24.10
opkg install /tmp/failsafe-dns-proxy_*.ipk
opkg install /tmp/luci-app-failsafe-dns-proxy_*.ipk
opkg install /tmp/luci-i18n-failsafe-dns-proxy-ru_*.ipk

# OpenWrt 25.12
apk --allow-untrusted add /tmp/failsafe-dns-proxy-*.apk
apk --allow-untrusted add /tmp/luci-app-failsafe-dns-proxy-*.apk
apk --allow-untrusted add /tmp/luci-i18n-failsafe-dns-proxy-ru-*.apk
```

**Build one OpenWrt target** produces a build artifact, not an official project
release. Use manual package installation and test on your own device. A
successful compile confirms SDK and package-format compatibility, not runtime
behavior on a specific router.

After installation, follow [Quick start](docs/en/quick-start.md).

## Contributing

Contributions are welcome: code, docs, packaging, translations, bug reports, and
real-device testing. The project stays useful when people extend it, fix rough
edges, and share results from routers that are not in the default release
matrix yet.

If you successfully ran the daemon on your hardware, please open a pull request
or issue with device model, OpenWrt version, target/subtarget, and what you
verified (install, failover, failback, reload). That helps others choose a
proven combination.

Details: [Contributing guide](docs/en/contributing.md).
