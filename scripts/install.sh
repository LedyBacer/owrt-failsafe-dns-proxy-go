#!/bin/sh

set -e

. /usr/share/libubox/jshn.sh

manifest=""
base_url=""
source_dir=""
configure_dnsmasq=0
include_luci=1
include_ru=1

usage() {
	cat <<'EOF'
usage: install.sh --manifest FILE_OR_URL [options]

Options:
  --base-url URL          base URL for package files
  --source-dir DIRECTORY  read package files locally instead of downloading
  --configure-dnsmasq     explicitly enable verified dnsmasq integration
  --daemon-only           install only the daemon package
  --no-russian            skip Russian localization
EOF
}

detect_openwrt() {
	command -v ubus >/dev/null 2>&1 || {
		printf '%s\n' "ubus is required" >&2
		return 1
	}
	command -v jsonfilter >/dev/null 2>&1 || {
		printf '%s\n' "jsonfilter is required" >&2
		return 1
	}
	local board target_pair
	board="$(ubus call system board)" || return 1
	OPENWRT_VERSION="$(printf '%s' "$board" | jsonfilter -e '@.release.version')"
	target_pair="$(printf '%s' "$board" | jsonfilter -e '@.release.target')"
	OPENWRT_TARGET="${target_pair%%/*}"
	OPENWRT_SUBTARGET="${target_pair#*/}"
	if command -v opkg >/dev/null 2>&1; then
		PACKAGE_MANAGER=opkg
		OPENWRT_PKGARCH="$(opkg print-architecture |
			awk '$1 == "arch" && $2 != "all" && $2 != "noarch" { arch=$2 } END { print arch }')"
	elif command -v apk >/dev/null 2>&1; then
		PACKAGE_MANAGER=apk
		OPENWRT_PKGARCH="$(apk --print-arch)"
	else
		printf '%s\n' "neither opkg nor apk was found" >&2
		return 1
	fi
	[ -n "$OPENWRT_VERSION" ] &&
		[ -n "$OPENWRT_TARGET" ] &&
		[ -n "$OPENWRT_SUBTARGET" ] &&
		[ -n "$OPENWRT_PKGARCH" ]
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--manifest) manifest="$2"; shift 2 ;;
		--base-url) base_url="${2%/}"; shift 2 ;;
		--source-dir) source_dir="$2"; shift 2 ;;
		--configure-dnsmasq) configure_dnsmasq=1; shift ;;
		--daemon-only) include_luci=0; include_ru=0; shift ;;
		--no-russian) include_ru=0; shift ;;
		-h|--help) usage; exit 0 ;;
		*) usage >&2; exit 2 ;;
	esac
done

[ -n "$manifest" ] || {
	usage >&2
	exit 2
}
[ -z "$source_dir" ] || [ -d "$source_dir" ] || {
	printf 'source directory does not exist: %s\n' "$source_dir" >&2
	exit 2
}

temporary="$(mktemp -d /tmp/failsafe-dns-proxy-install.XXXXXX)"
trap 'rm -rf "$temporary"' EXIT INT TERM

fetch() {
	local source="$1" output="$2"
	case "$source" in
		http://*|https://*)
			if command -v uclient-fetch >/dev/null 2>&1; then
				uclient-fetch -q -O "$output" "$source"
			else
				wget -q -O "$output" "$source"
			fi
			;;
		*) cp "$source" "$output" ;;
	esac
}

manifest_file="$temporary/manifest.json"
fetch "$manifest" "$manifest_file"
detect_openwrt

daemon_file=""
daemon_sha=""
luci_file=""
luci_sha=""
ru_file=""
ru_sha=""

json_load_file "$manifest_file"
json_select artifacts
artifact_keys=""
json_get_keys artifact_keys
for artifact_key in $artifact_keys; do
	json_select "$artifact_key"
	openwrt_version=""
	target=""
	subtarget=""
	pkgarch=""
	package_manager=""
	json_get_vars openwrt_version target subtarget pkgarch package_manager
	if [ "$openwrt_version" = "$OPENWRT_VERSION" ] &&
		[ "$target" = "$OPENWRT_TARGET" ] &&
		[ "$subtarget" = "$OPENWRT_SUBTARGET" ] &&
		[ "$pkgarch" = "$OPENWRT_PKGARCH" ] &&
		[ "$package_manager" = "$PACKAGE_MANAGER" ]; then
		json_select daemon
		json_get_var daemon_file file
		json_get_var daemon_sha sha256
		json_select ..
		json_select luci
		json_get_var luci_file file
		json_get_var luci_sha sha256
		json_select ..
		json_select i18n_ru
		json_get_var ru_file file
		json_get_var ru_sha sha256
		json_select ..
	fi
	json_select ..
done
json_cleanup

[ -n "$daemon_file" ] && [ -n "$daemon_sha" ] || {
	printf 'no exact artifact for OpenWrt %s %s/%s %s (%s)\n' \
		"$OPENWRT_VERSION" "$OPENWRT_TARGET" "$OPENWRT_SUBTARGET" "$OPENWRT_PKGARCH" "$PACKAGE_MANAGER" >&2
	exit 3
}
if [ "$include_luci" -eq 1 ] && { [ -z "$luci_file" ] || [ -z "$luci_sha" ]; }; then
	printf '%s\n' "the exact manifest entry does not contain a LuCI package" >&2
	exit 3
fi
if [ "$include_ru" -eq 1 ] && { [ -z "$ru_file" ] || [ -z "$ru_sha" ]; }; then
	printf '%s\n' "the exact manifest entry does not contain Russian localization" >&2
	exit 3
fi

download_package() {
	local file="$1" sha="$2" source output
	[ -n "$file" ] && [ -n "$sha" ] || return 0
	output="$temporary/$file"
	if [ -n "$source_dir" ]; then
		source="$source_dir/$file"
	else
		[ -n "$base_url" ] || {
			printf '%s\n' "--base-url or --source-dir is required for package files" >&2
			exit 2
		}
		source="$base_url/$file"
	fi
	fetch "$source" "$output"
	printf '%s  %s\n' "$sha" "$output" | sha256sum -c - >/dev/null
}

download_package "$daemon_file" "$daemon_sha"
if [ "$include_luci" -eq 1 ]; then
	download_package "$luci_file" "$luci_sha"
fi
if [ "$include_ru" -eq 1 ]; then
	download_package "$ru_file" "$ru_sha"
fi

set -- "$temporary/$daemon_file"
[ "$include_luci" -eq 0 ] || set -- "$@" "$temporary/$luci_file"
[ "$include_ru" -eq 0 ] || set -- "$@" "$temporary/$ru_file"

case "$PACKAGE_MANAGER" in
	opkg) opkg install "$@" ;;
	apk) apk --allow-untrusted add "$@" ;;
esac

if [ "$configure_dnsmasq" -eq 1 ]; then
	/usr/sbin/failsafe-dns-proxy-dnsmasq enable
fi

printf 'Failsafe DNS Proxy installation completed for %s %s/%s %s.\n' \
	"$OPENWRT_VERSION" "$OPENWRT_TARGET" "$OPENWRT_SUBTARGET" "$OPENWRT_PKGARCH"
