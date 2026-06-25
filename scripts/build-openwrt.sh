#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat <<'EOF'
usage: build-openwrt.sh OPENWRT_VERSION TARGET SUBTARGET [OUTPUT_DIR]

Builds the daemon, LuCI application and Russian localization with the exact
official OpenWrt SDK. The SDK filename and SHA-256 are discovered from the
official target sha256sums file.
EOF
}

if [[ $# -lt 3 || $# -gt 4 ]]; then
	usage >&2
	exit 2
fi

readonly OPENWRT_VERSION="$1"
readonly TARGET="$2"
readonly SUBTARGET="$3"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output_dir="${4:-${FDP_OUTPUT_DIR:-${repo_dir}/dist}}"
work_dir="${FDP_BUILD_DIR:-${repo_dir}/build/openwrt-${OPENWRT_VERSION}-${TARGET}-${SUBTARGET}}"
download_dir="${FDP_DOWNLOAD_DIR:-${repo_dir}/build/downloads}"
target_url="https://downloads.openwrt.org/releases/${OPENWRT_VERSION}/targets/${TARGET}/${SUBTARGET}"

if [[ "$(uname -s)-$(uname -m)" != "Linux-x86_64" ]]; then
	echo "The official OpenWrt SDK requires Linux x86_64 (current: $(uname -s)-$(uname -m))." >&2
	exit 2
fi

# OpenWrt builds each required Go toolchain from source. Prevent bootstrap Go
# binaries from interpreting nested go.mod files as a request to download an
# intermediate toolchain, which is unavailable for versions such as go1.23.0.
export GOTOOLCHAIN=local

mkdir -p "$download_dir" "$work_dir" "$output_dir"
sha256sums="${work_dir}/sha256sums"
profiles="${work_dir}/profiles.json"
curl --fail --location --silent --show-error --output "$sha256sums" "${target_url}/sha256sums"
curl --fail --location --silent --show-error --output "$profiles" "${target_url}/profiles.json"

sdk_line="$(grep -E " [*]?openwrt-sdk-${OPENWRT_VERSION}-${TARGET}-${SUBTARGET//\//-}_[^ ]+\\.Linux-x86_64\\.tar\\.(zst|xz)$" "$sha256sums" | head -n 1)"
if [[ -z "$sdk_line" ]]; then
	echo "No official Linux x86_64 SDK found for ${OPENWRT_VERSION} ${TARGET}/${SUBTARGET}." >&2
	exit 3
fi
sdk_sha256="${sdk_line%% *}"
sdk_file="${sdk_line#* }"
sdk_file="${sdk_file#\*}"
archive="${download_dir}/${sdk_file}"
sdk_url="${target_url}/${sdk_file}"

if [[ ! -f "$archive" ]]; then
	curl --fail --location --output "$archive" "$sdk_url"
fi
echo "${sdk_sha256}  ${archive}" | sha256sum --check -

sdk_dir="${work_dir}/sdk"
sdk_stamp="${sdk_dir}/.fdp-sdk-sha256"
if [[ ! -f "${sdk_dir}/Makefile" || ! -f "$sdk_stamp" || "$(cat "$sdk_stamp")" != "$sdk_sha256" ]]; then
	rm -rf "$sdk_dir"
	mkdir -p "$sdk_dir"
	case "$archive" in
		*.tar.zst) tar --use-compress-program=unzstd -xf "$archive" -C "$sdk_dir" --strip-components=1 ;;
		*.tar.xz) tar -xJf "$archive" -C "$sdk_dir" --strip-components=1 ;;
	esac
	printf '%s\n' "$sdk_sha256" >"$sdk_stamp"
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

# The OpenWrt Go feed intentionally unexports GOTOOLCHAIN. Some bootstrap and
# host Makefiles do not add it back, so nested Go source modules can trigger an
# impossible automatic download of an intermediate toolchain (for example
# go1.23.0 while building Go 1.24). Patch every explicit GOENV=off command
# environment to keep the SDK build fully local and reproducible.
mapfile -t go_makefiles < <(
	find feeds/packages/lang/golang -type f \( -name Makefile -o -name '*.mk' \) -print
)
python3 "${repo_dir}/scripts/patch-openwrt-go-env.py" "${go_makefiles[@]}"

rm -rf package/failsafe-dns-proxy package/luci-app-failsafe-dns-proxy
ln -s "${repo_dir}/package/failsafe-dns-proxy" package/failsafe-dns-proxy
ln -s "${repo_dir}/package/luci-app-failsafe-dns-proxy" package/luci-app-failsafe-dns-proxy

cat >.config <<EOF
CONFIG_TARGET_${TARGET}=y
CONFIG_TARGET_${TARGET}_${SUBTARGET//-/_}=y
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

case "$OPENWRT_VERSION" in
	24.10.*) package_extension=ipk; package_manager=opkg ;;
	25.12.*) package_extension=apk; package_manager=apk ;;
	*)
		echo "Unsupported package format for OpenWrt ${OPENWRT_VERSION}." >&2
		exit 3
		;;
esac

rm -rf "$output_dir"
mkdir -p "$output_dir"
find bin/packages -type f \( \
	-name "failsafe-dns-proxy_*.${package_extension}" -o \
	-name "failsafe-dns-proxy-*.${package_extension}" -o \
	-name "luci-app-failsafe-dns-proxy_*.${package_extension}" -o \
	-name "luci-app-failsafe-dns-proxy-*.${package_extension}" -o \
	-name "luci-i18n-failsafe-dns-proxy-ru_*.${package_extension}" -o \
	-name "luci-i18n-failsafe-dns-proxy-ru-*.${package_extension}" \
\) -exec cp -v {} "$output_dir/" \;

daemon_artifact="$(find "$output_dir" -maxdepth 1 -type f -name "failsafe-dns-proxy*.${package_extension}" ! -name 'failsafe-dns-proxy-soak*' -print -quit)"
luci_artifact="$(find "$output_dir" -maxdepth 1 -type f -name "luci-app-failsafe-dns-proxy*.${package_extension}" -print -quit)"
i18n_artifact="$(find "$output_dir" -maxdepth 1 -type f -name "luci-i18n-failsafe-dns-proxy-ru*.${package_extension}" -print -quit)"
test -n "$daemon_artifact"
test -n "$luci_artifact"
test -n "$i18n_artifact"

daemon_pkgdir="$(find build_dir -type d -path '*/.pkgdir/failsafe-dns-proxy' -print -quit)"
luci_pkgdir="$(find build_dir -type d -path '*/.pkgdir/luci-app-failsafe-dns-proxy' -print -quit)"
i18n_pkgdir="$(find build_dir -type d -path '*/.pkgdir/luci-i18n-failsafe-dns-proxy-ru' -print -quit)"
test -x "$daemon_pkgdir/usr/sbin/failsafe-dns-proxy"
test -x "$daemon_pkgdir/usr/sbin/failsafe-dns-proxy-dnsmasq"
test -x "$daemon_pkgdir/usr/sbin/failsafe-dns-proxy-soak"
test -f "$daemon_pkgdir/etc/config/failsafe-dns-proxy"
test -x "$daemon_pkgdir/etc/init.d/failsafe-dns-proxy"
test -f "$luci_pkgdir/www/luci-static/resources/view/failsafe-dns-proxy/overview.js"
test -x "$luci_pkgdir/usr/libexec/rpcd/luci.failsafe-dns-proxy"
find "$i18n_pkgdir" -type f -name '*.lmo' | grep -q .

if [[ "$package_extension" == ipk ]]; then
	smoke_dir="$(mktemp -d)"
	trap 'rm -rf "$smoke_dir"' EXIT
	for pair in "daemon:$daemon_artifact" "luci:$luci_artifact" "i18n:$i18n_artifact"; do
		name="${pair%%:*}"
		artifact="${pair#*:}"
		mkdir -p "$smoke_dir/$name" "$smoke_dir/$name-data"
		tar -xzf "$artifact" -C "$smoke_dir/$name"
		tar -xzf "$smoke_dir/$name/data.tar.gz" -C "$smoke_dir/$name-data"
	done
	test -x "$smoke_dir/daemon-data/usr/sbin/failsafe-dns-proxy"
	test -x "$smoke_dir/daemon-data/usr/sbin/failsafe-dns-proxy-dnsmasq"
	test -x "$smoke_dir/daemon-data/usr/sbin/failsafe-dns-proxy-soak"
	test -f "$smoke_dir/daemon-data/etc/config/failsafe-dns-proxy"
	test -x "$smoke_dir/daemon-data/etc/init.d/failsafe-dns-proxy"
	test -f "$smoke_dir/luci-data/www/luci-static/resources/view/failsafe-dns-proxy/overview.js"
	test -x "$smoke_dir/luci-data/usr/libexec/rpcd/luci.failsafe-dns-proxy"
	find "$smoke_dir/i18n-data" -type f -name '*.lmo' | grep -q .
fi

pkgarch="$(python3 - "$profiles" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as handle:
    data = json.load(handle)
print(data["arch_packages"])
PY
)"

(cd "$output_dir" && sha256sum ./*."$package_extension" >SHA256SUMS)
python3 "${repo_dir}/scripts/generate-build-metadata.py" \
	--output "${output_dir}/build-metadata.json" \
	--openwrt-version "$OPENWRT_VERSION" \
	--target "$TARGET" \
	--subtarget "$SUBTARGET" \
	--pkgarch "$pkgarch" \
	--package-manager "$package_manager" \
	--daemon "$daemon_artifact" \
	--luci "$luci_artifact" \
	--i18n-ru "$i18n_artifact"
