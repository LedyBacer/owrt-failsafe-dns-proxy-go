# Troubleshooting

**English** | [Русский](../ru/troubleshooting.md)

## Service does not start

```sh
failsafe-dns-proxy check-config
logread -e failsafe-dns-proxy
```

Check `enabled=1`, unique priorities, upstream IP addresses, and timeout
relationships.

## Status socket unavailable

Verify the daemon is running and `status_socket` matches the CLI:

```sh
failsafe-dns-proxy status \
  --socket /var/run/failsafe-dns-proxy.sock
```

## Clients bypass the proxy after configuration

Check dnsmasq:

```sh
uci show dhcp.@dnsmasq[0].noresolv
uci show dhcp.@dnsmasq[0].server
failsafe-dns-proxy-dnsmasq status
```

For the strict layout you need `noresolv=1`; otherwise dnsmasq may use DNS
from DHCP in parallel.

## Service intermittently stalls after pointing dnsmasq at it

If the logs show timeouts to a local upstream such as `127.0.0.1:5053` or
`127.0.0.1:5054`, and that port belongs to `https-dns-proxy`, make sure
`https-dns-proxy` no longer rewrites dnsmasq itself:

```sh
uci -q get https-dns-proxy.config.dnsmasq_config_update
uci set https-dns-proxy.config.dnsmasq_config_update='-'
uci commit https-dns-proxy
/etc/init.d/https-dns-proxy restart
failsafe-dns-proxy-dnsmasq dry-run
failsafe-dns-proxy-dnsmasq enable
```

`https-dns-proxy` has its own dnsmasq integration enabled by default. If both
that integration and the Failsafe DNS Proxy integration manage
`/etc/config/dhcp`, the local DoH upstream can start timing out after a service
restart, WAN event, or heartbeat.

## Fallback is too slow

Reduce `attempt_timeout_ms` carefully and only after measuring. A value that is
too small creates false failures on a busy router or slow link.

[← Documentation index](index.md)
