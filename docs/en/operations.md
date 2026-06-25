# Operations

**English** | [Русский](../ru/operations.md)

## Common commands

```sh
failsafe-dns-proxy version
failsafe-dns-proxy check-config
failsafe-dns-proxy status
failsafe-dns-proxy status --json
failsafe-dns-proxy probe encrypted_local
failsafe-dns-proxy probe --json public_fallback
failsafe-dns-proxy self-test

/etc/init.d/failsafe-dns-proxy status
/etc/init.d/failsafe-dns-proxy reload
logread -e failsafe-dns-proxy
```

The daemon does not log every DNS query. The log contains operational events:
failure, recovery transition, restoration, failover/failback, rejected reload,
and emergency mode.

## Multi-day soak test

```sh
failsafe-dns-proxy-soak start 168 60
failsafe-dns-proxy-soak status
failsafe-dns-proxy-soak report
failsafe-dns-proxy-soak stop
```

The first argument is duration in hours; the second is the sampling interval
in seconds. The monitor records RSS, goroutines, file descriptors, heap, active
upstream, and self-test results in `/tmp/failsafe-dns-proxy-soak`.

## Disable dnsmasq integration

```sh
failsafe-dns-proxy-dnsmasq disable
```

The command restores saved parameters. If the `dhcp` UCI package was changed
after integration was enabled, automatic restore is rejected so newer settings
are not overwritten.

[← Documentation index](index.md)
