# Quick start

**English** | [Русский](../ru/quick-start.md)

1. Open LuCI → **Services → Failsafe DNS Proxy** or edit
   `/etc/config/failsafe-dns-proxy`.
1. Replace the example upstreams with your own addresses.
1. Enable the daemon and verify the configuration:

```sh
uci set failsafe-dns-proxy.main.enabled='1'
uci commit failsafe-dns-proxy

failsafe-dns-proxy check-config
/etc/init.d/failsafe-dns-proxy enable
/etc/init.d/failsafe-dns-proxy start
failsafe-dns-proxy self-test
failsafe-dns-proxy status
```

1. Only after a successful self-test, point dnsmasq at the proxy:

```sh
failsafe-dns-proxy-dnsmasq dry-run
failsafe-dns-proxy-dnsmasq enable
failsafe-dns-proxy-dnsmasq status
```

The `enable` command saves the previous dnsmasq UCI configuration, applies
`noresolv=1` and `server=127.0.0.1#5359`, verifies DNS, and rolls back on
error.

[← Documentation index](index.md)
