#!/bin/sh

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
		OPENWRT_PKGARCH="$(opkg print-architecture | awk '$1 == "arch" && $2 != "all" && $2 != "noarch" { arch=$2 } END { print arch }')"
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
