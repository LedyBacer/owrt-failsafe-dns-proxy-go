# Configuration

**English** | [Русский](../ru/configuration.md)

## Defaults shipped with the package

```uci
config main 'main'
        option enabled '0'
        option listen_addr '127.0.0.1'
        option listen_port '5359'
        option attempt_timeout_ms '700'
        option request_timeout_ms '2000'
        option health_interval_s '5'
        option fail_threshold '2'
        option recover_threshold '2'
        option max_concurrent '128'
        option status_socket '/var/run/failsafe-dns-proxy.sock'
        list probe 'example.com:A'
        list probe 'ya.ru:A'

config upstream 'encrypted_local'
        option enabled '1'
        option priority '10'
        option protocol 'udp'
        option address '127.0.0.1'
        option port '5054'

config upstream 'public_fallback'
        option enabled '1'
        option priority '20'
        option protocol 'udp'
        option address '77.88.8.8'
        option port '53'
```

`enabled` is intentionally `0`: the package must not start an unverified
configuration automatically.

On the test AX6S the same layout is used, but the daemon is enabled:
`failsafe-dns-proxy.main.enabled=1`. The primary upstream is local
`https-dns-proxy` on `127.0.0.1:5054`; the fallback is `77.88.8.8:53`.

## Parameter reference

| Parameter | Default | Purpose |
| --- | ---: | --- |
| `listen_addr` | `127.0.0.1` | listener address; loopback is safe with dnsmasq |
| `listen_port` | `5359` | UDP and TCP proxy port |
| `attempt_timeout_ms` | `700` | maximum time for one upstream attempt |
| `request_timeout_ms` | `2000` | total request budget including all fallbacks |
| `health_interval_s` | `5` | base active probe interval |
| `fail_threshold` | `2` | confirmed failures before `down` |
| `recover_threshold` | `2` | successful probes before recovery |
| `max_concurrent` | `128` | global concurrent request limit |
| `probe` | two DNS questions | background transport health checks |
| `priority` | `10`, `20` | lower number means higher priority |
| `protocol` | `udp` | plain DNS transport; `udp` and `tcp` are supported |

`request_timeout_ms` must be at least `attempt_timeout_ms`. With multiple
upstreams, the total budget should still allow at least one fallback attempt.

## Changing settings

### Via LuCI

Open **Services → Failsafe DNS Proxy**. The interface lets you:

- enable the service and change the listener;
- add, disable, or remove upstreams;
- change priority, protocol, IP, and port;
- adjust timeouts, thresholds, and probe questions;
- check configuration and individual upstreams;
- view the active upstream, health state, and recent errors;
- enable or disable dnsmasq integration.

### Via UCI

Replace the fallback DNS:

```sh
uci set failsafe-dns-proxy.public_fallback.address='1.1.1.1'
uci set failsafe-dns-proxy.public_fallback.port='53'
uci commit failsafe-dns-proxy

failsafe-dns-proxy check-config
/etc/init.d/failsafe-dns-proxy reload
```

Add another upstream:

```sh
uci set failsafe-dns-proxy.last_resort='upstream'
uci set failsafe-dns-proxy.last_resort.enabled='1'
uci set failsafe-dns-proxy.last_resort.priority='30'
uci set failsafe-dns-proxy.last_resort.protocol='tcp'
uci set failsafe-dns-proxy.last_resort.address='8.8.8.8'
uci set failsafe-dns-proxy.last_resort.port='53'
uci commit failsafe-dns-proxy

failsafe-dns-proxy check-config
/etc/init.d/failsafe-dns-proxy reload
```

Upstream names and `priority` values must be unique. Upstream addresses must
currently be IP addresses. A hostname can create a bootstrap loop if dnsmasq
already points at this proxy.

Reload is atomic. Invalid configuration is rejected and the daemon keeps the
previous one. Changing `listen_addr`, `listen_port`, or `status_socket`
requires a restart:

```sh
/etc/init.d/failsafe-dns-proxy restart
```

[← Documentation index](index.md)
