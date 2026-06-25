# Limitations

**English** | [Русский](../ru/limitations.md)

- Native DoH, DoT, and DoQ are not implemented yet.
- Upstream must be an IP address; loop-safe hostname bootstrap is absent.
- No DNS cache, filtering, DNSSEC validation, split DNS, or load balancing.
- dnsmasq remains required for LAN cache and local names.
- An APK build does not mean hardware validation on every target.
- Official OpenWrt is supported; forks require separate validation.

[← Documentation index](index.md)
