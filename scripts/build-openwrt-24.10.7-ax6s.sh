#!/usr/bin/env bash
set -euo pipefail

readonly OPENWRT_VERSION=24.10.7
readonly TARGET=mediatek
readonly SUBTARGET=mt7622
readonly SDK_FILE=openwrt-sdk-24.10.7-mediatek-mt7622_gcc-13.3.0_musl.Linux-x86_64.tar.zst
readonly SDK_SHA256=326064cd8da2b6c9dd254748a6130d2c48a10efb9624d6457ee6c394e9ad62dd
readonly SDK_URL="https://downloads.openwrt.org/releases/${OPENWRT_VERSION}/targets/${TARGET}/${SUBTARGET}/${SDK_FILE}"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work_dir="${FDP_BUILD_DIR:-${repo_dir}/build/openwrt-${OPENWRT_VERSION}-${TARGET}-${SUBTARGET}}"
download_dir="${FDP_DOWNLOAD_DIR:-${repo_dir}/build/downloads}"
archive="${download_dir}/${SDK_FILE}"

if [[ "$(uname -s)-$(uname -m)" != "Linux-x86_64" ]]; then
	echo "This official SDK requires a Linux x86_64 host (current: $(uname -s)-$(uname -m))." >&2
	echo "Run this script on Linux x86_64 or in a Linux x86_64 CI runner." >&2
	exit 2
fi

mkdir -p "$download_dir" "$work_dir"
if [[ ! -f "$archive" ]]; then
	curl --fail --location --output "$archive" "$SDK_URL"
fi
echo "${SDK_SHA256}  ${archive}" | sha256sum --check -

sdk_dir="${work_dir}/sdk"
if [[ ! -f "${sdk_dir}/Makefile" ]]; then
	rm -rf "$sdk_dir"
	mkdir -p "$sdk_dir"
	tar --use-compress-program=unzstd -xf "$archive" -C "$sdk_dir" --strip-components=1
fi

cd "$sdk_dir"
if [[ -d feeds/base/.git && -d feeds/packages/.git && -d feeds/luci/.git ]]; then
	./scripts/feeds update -i base packages luci
else
	./scripts/feeds update base packages luci
fi
./scripts/feeds install ucode
./scripts/feeds install rpcd
./scripts/feeds install golang
./scripts/feeds install luci-base
./scripts/feeds install csstidy
./scripts/feeds install luasrcdiet
rm -rf package/failsafe-dns-proxy
ln -s "${repo_dir}/package/failsafe-dns-proxy" package/failsafe-dns-proxy
rm -rf package/luci-app-failsafe-dns-proxy
ln -s "${repo_dir}/package/luci-app-failsafe-dns-proxy" package/luci-app-failsafe-dns-proxy

cat > .config <<'EOF'
CONFIG_TARGET_mediatek=y
CONFIG_TARGET_mediatek_mt7622=y
CONFIG_PACKAGE_failsafe-dns-proxy=m
CONFIG_PACKAGE_luci-app-failsafe-dns-proxy=m
CONFIG_PACKAGE_luci-i18n-failsafe-dns-proxy-ru=m
# CONFIG_PACKAGE_liblucihttp-lua is not set
EOF
make defconfig
make package/failsafe-dns-proxy/clean FDP_SOURCE_DIR="$repo_dir"
make package/failsafe-dns-proxy/compile V=s FDP_SOURCE_DIR="$repo_dir"
make package/luci-app-failsafe-dns-proxy/clean
make package/luci-app-failsafe-dns-proxy/compile V=s

mkdir -p "${repo_dir}/dist"
rm -f "${repo_dir}/dist"/failsafe-dns-proxy_*.ipk
rm -f "${repo_dir}/dist"/luci-app-failsafe-dns-proxy_*.ipk
rm -f "${repo_dir}/dist"/luci-i18n-failsafe-dns-proxy-ru_*.ipk
find bin/packages -type f \( \
	-name 'failsafe-dns-proxy_*.ipk' -o \
	-name 'luci-app-failsafe-dns-proxy_*.ipk' -o \
	-name 'luci-i18n-failsafe-dns-proxy-ru_*.ipk' \
\) -exec cp -v {} "${repo_dir}/dist/" \;

daemon_artifact="$(find "${repo_dir}/dist" -maxdepth 1 -name 'failsafe-dns-proxy_*.ipk' -print -quit)"
luci_artifact="$(find "${repo_dir}/dist" -maxdepth 1 -name 'luci-app-failsafe-dns-proxy_*.ipk' -print -quit)"
i18n_artifact="$(find "${repo_dir}/dist" -maxdepth 1 -name 'luci-i18n-failsafe-dns-proxy-ru_*.ipk' -print -quit)"
test -n "$daemon_artifact"
test -n "$luci_artifact"
test -n "$i18n_artifact"

smoke_dir="$(mktemp -d)"
trap 'rm -rf "$smoke_dir"' EXIT
mkdir -p "$smoke_dir/daemon" "$smoke_dir/luci" "$smoke_dir/i18n"
tar -xzf "$daemon_artifact" -C "$smoke_dir/daemon"
mkdir -p "$smoke_dir/data"
tar -xzf "$smoke_dir/daemon/data.tar.gz" -C "$smoke_dir/data"
test -x "$smoke_dir/data/usr/sbin/failsafe-dns-proxy"
test -x "$smoke_dir/data/usr/sbin/failsafe-dns-proxy-dnsmasq"
test -x "$smoke_dir/data/usr/sbin/failsafe-dns-proxy-soak"
test -f "$smoke_dir/data/etc/config/failsafe-dns-proxy"
test -x "$smoke_dir/data/etc/init.d/failsafe-dns-proxy"
file "$smoke_dir/data/usr/sbin/failsafe-dns-proxy" | grep -q 'ARM aarch64'
file "$smoke_dir/data/usr/sbin/failsafe-dns-proxy" | grep -q 'statically linked'

tar -xzf "$luci_artifact" -C "$smoke_dir/luci"
mkdir -p "$smoke_dir/luci-data"
tar -xzf "$smoke_dir/luci/data.tar.gz" -C "$smoke_dir/luci-data"
test -f "$smoke_dir/luci-data/www/luci-static/resources/view/failsafe-dns-proxy/overview.js"
test -x "$smoke_dir/luci-data/usr/libexec/rpcd/luci.failsafe-dns-proxy"
test -f "$smoke_dir/luci-data/usr/share/rpcd/acl.d/luci-app-failsafe-dns-proxy.json"
test -f "$smoke_dir/luci-data/usr/share/luci/menu.d/luci-app-failsafe-dns-proxy.json"

tar -xzf "$i18n_artifact" -C "$smoke_dir/i18n"
mkdir -p "$smoke_dir/i18n-data"
tar -xzf "$smoke_dir/i18n/data.tar.gz" -C "$smoke_dir/i18n-data"
find "$smoke_dir/i18n-data" -type f -name '*.lmo' | grep -q .

(cd "${repo_dir}/dist" && sha256sum ./*.ipk > SHA256SUMS)
