#!/bin/sh

set -eu

repo_dir="$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)"
temporary="$(mktemp -d)"
trap 'rm -rf "$temporary"' EXIT INT TERM
mock="$temporary/mock"
state="$temporary/state"
mkdir -p "$mock" "$state"

cat >"$mock/uci" <<'EOF'
#!/bin/sh
state="${MOCK_STATE:?}"
quiet=0
[ "${1:-}" != "-q" ] || { quiet=1; shift; }
command="$1"
shift
case "$command" in
	get)
		case "$1" in
			failsafe-dns-proxy.main.listen_addr) echo 127.0.0.1 ;;
			failsafe-dns-proxy.main.listen_port) echo 5359 ;;
			failsafe-dns-proxy.main.probe) echo "example.com:A ya.ru:A" ;;
			failsafe-dns-proxy.encrypted_local.enabled) echo 1 ;;
			failsafe-dns-proxy.encrypted_local.address) echo 127.0.0.1 ;;
			failsafe-dns-proxy.encrypted_local.port) echo 5054 ;;
			failsafe-dns-proxy.public_fallback.enabled) echo 1 ;;
			failsafe-dns-proxy.public_fallback.address) echo 77.88.8.8 ;;
			failsafe-dns-proxy.public_fallback.port) echo 53 ;;
			*) [ "$quiet" -eq 1 ] || echo "unknown get $1" >&2; exit 1 ;;
		esac
		;;
	show)
		case "$1" in
			dhcp)
				echo "dhcp.cfg=dnsmasq"
				;;
			failsafe-dns-proxy)
				echo "failsafe-dns-proxy.main=main"
				echo "failsafe-dns-proxy.encrypted_local=upstream"
				echo "failsafe-dns-proxy.public_fallback=upstream"
				;;
		esac
		;;
	export)
		if [ -f "$state/mode" ] && [ "$(cat "$state/mode")" = applied ]; then
			printf "package dhcp\n\nconfig dnsmasq 'cfg'\n\toption noresolv '1'\n\tlist server '127.0.0.1#5359'\n"
		else
			printf "package dhcp\n\nconfig dnsmasq 'cfg'\n\toption noresolv '0'\n\tlist server '1.1.1.1'\n"
		fi
		;;
	batch)
		cat >"$state/batch"
		echo applied >"$state/mode"
		;;
	import)
		cat >"$state/import"
		echo restored >"$state/mode"
		;;
	commit|revert)
		;;
	*)
		echo "unsupported mock uci command: $command" >&2
		exit 1
		;;
esac
EOF

cat >"$mock/proxy" <<'EOF'
#!/bin/sh
exit 0
EOF

cat >"$mock/init" <<'EOF'
#!/bin/sh
case "$1" in
	running|restart) exit 0 ;;
	*) exit 1 ;;
esac
EOF

cat >"$mock/nslookup" <<'EOF'
#!/bin/sh
[ "${MOCK_NSLOOKUP_FAIL:-0}" = 0 ]
EOF

cat >"$mock/timeout" <<'EOF'
#!/bin/sh
shift
exec "$@"
EOF

chmod +x "$mock"/*

export MOCK_STATE="$state"
export FDP_PROXY_CONFIG="$temporary/proxy.conf"
export FDP_STATE_DIR="$temporary/integration"
export FDP_BACKUP_FILE="$temporary/integration/dhcp.backup"
export FDP_MARKER_FILE="$temporary/integration/enabled"
export FDP_UCI_BIN="$mock/uci"
export FDP_PROXY_BIN="$mock/proxy"
export FDP_DNSMASQ_INIT="$mock/init"
export FDP_PROXY_INIT="$mock/init"
export FDP_NSLOOKUP_BIN="$mock/nslookup"
export FDP_TIMEOUT_BIN="$mock/timeout"
export FDP_VERIFY_ATTEMPTS=1

helper="$repo_dir/package/failsafe-dns-proxy/files/usr/sbin/failsafe-dns-proxy-dnsmasq"

"$helper" dry-run >/dev/null
[ ! -e "$FDP_BACKUP_FILE" ]

"$helper" enable >/dev/null
[ -s "$FDP_BACKUP_FILE" ]
[ -s "$FDP_MARKER_FILE" ]
grep -q "add_list dhcp.cfg.server='127.0.0.1#5359'" "$state/batch"

echo changed >"$state/mode"
if "$helper" disable >/dev/null 2>&1; then
	echo "disable unexpectedly overwrote a changed DHCP configuration" >&2
	exit 1
fi
[ -s "$FDP_MARKER_FILE" ]
echo applied >"$state/mode"

"$helper" disable >/dev/null
[ ! -e "$FDP_BACKUP_FILE" ]
[ ! -e "$FDP_MARKER_FILE" ]
grep -q "config dnsmasq 'cfg'" "$state/import"

export MOCK_NSLOOKUP_FAIL=1
if "$helper" enable >/dev/null 2>&1; then
	echo "enable unexpectedly succeeded with failed verification" >&2
	exit 1
fi
[ ! -e "$FDP_BACKUP_FILE" ]
[ ! -e "$FDP_MARKER_FILE" ]
[ "$(cat "$state/mode")" = restored ]

printf '%s\n' "shell integration tests passed"
