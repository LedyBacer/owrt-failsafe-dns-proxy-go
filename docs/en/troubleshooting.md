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

## Fallback is too slow

Reduce `attempt_timeout_ms` carefully and only after measuring. A value that is
too small creates false failures on a busy router or slow link.

[← Documentation index](index.md)
