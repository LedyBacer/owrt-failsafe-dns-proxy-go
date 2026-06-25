# Features

**English** | [Русский](../ru/features.md)

- UDP and TCP for clients and plain DNS upstreams;
- TCP retry after a truncated UDP response;
- strict numeric upstream priority;
- states `unknown`, `healthy`, `suspect`, `down`, and `recovering`;
- passive failure detection and background active probes;
- bounded per-attempt and total client request timeouts;
- recovery hysteresis and automatic failback;
- emergency half-open when every upstream is considered unavailable;
- DNS transaction ID, question, and response structure validation;
- atomic reload that keeps the previous configuration on error;
- UCI, procd, JSON status, LuCI, and Russian localization;
- explicit reversible dnsmasq integration;
- IPK for OpenWrt 24.10 and APK for OpenWrt 25.12;
- exact-match installer with SHA-256 verification.

[← Documentation index](index.md)
