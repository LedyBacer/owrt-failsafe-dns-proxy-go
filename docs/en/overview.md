# Overview

**English** | [Русский](../ru/overview.md)

## Problem

On OpenWrt, dnsmasq is the DNS server for your LAN: it answers clients, caches
responses, and resolves local hostnames. For upstream DNS it usually forwards to
one address — for example a local `https-dns-proxy`, `stubby`, or a public
resolver.

When that upstream stops responding, every device on the network loses DNS until
you notice and change the configuration. dnsmasq does not:

- try upstreams in a fixed priority order;
- skip an upstream that is already known to be down;
- switch back to the preferred resolver after it recovers.

Adding several `server=` lines to dnsmasq is not the same thing: you get
multiple resolvers, not strict primary/backup behavior with health tracking and
automatic failback.

## Solution

Failsafe DNS Proxy is a small local daemon that dnsmasq uses as its only
upstream. The daemon forwards queries to your real resolvers:

1. **Primary** — the resolver you prefer (often local encrypted DNS).
2. **Fallback** — plain DNS that should work even when the primary path fails.

The daemon picks the highest-priority working upstream, fails over within
bounded timeouts, remembers recent failures, and returns to the primary only
after background probes confirm recovery.

## What it is not

| This project | Not this |
| --- | --- |
| Failover layer between dnsmasq and upstreams | Replacement for dnsmasq |
| Plain UDP/TCP forwarding to configured resolvers | DoH/DoT/DoQ client |
| Health-aware upstream selection | DNS cache, ad blocking, or filtering |
| Explicit, reversible dnsmasq integration | Silent rewrite of router DNS settings |

If you only need one stable public resolver and never run a local DNS proxy,
you probably do not need this daemon — pointing dnsmasq directly at that
resolver is enough.

## Typical layout

```text
LAN clients
    |
    v
dnsmasq          <- clients, cache, local names
    |
    v
failsafe-dns-proxy 127.0.0.1:5359
    |-- priority 10 -> https-dns-proxy 127.0.0.1:5054
    `-- priority 20 -> fallback 77.88.8.8:53
```

See also [Features](features.md) and [Failover behavior](failover.md).

[← Documentation index](index.md)
