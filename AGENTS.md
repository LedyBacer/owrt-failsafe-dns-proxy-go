# AGENTS.md

## Project goal

Build a small, reliable DNS failover daemon for OpenWrt and a separate LuCI
application. The daemon must prefer upstreams in configured order, remember
temporarily failed upstreams, fail over within a bounded timeout, and return to
a higher-priority upstream only after confirmed recovery.

The supported OpenWrt baseline is 24.10.0. OpenWrt 24.10 uses `opkg`/IPK;
OpenWrt 25.12 and newer use `apk`/APK.

## Plan discipline

- `docs/PLAN.md` is the authoritative implementation roadmap and status
  ledger. Read it before starting any non-trivial change.
- Follow the order and priorities recorded in the plan unless the user
  explicitly changes them or a verified technical dependency requires
  reordering.
- Update `docs/PLAN.md` in the same patch whenever work is completed, added,
  removed, split, reprioritized, or found to be blocked.
- Mark tasks as complete only after the relevant implementation and
  proportionate verification are finished. Record partial completion and known
  limitations explicitly.
- Do not let README, package metadata, or claims of support get ahead of the
  status recorded in `docs/PLAN.md`.

## Product boundaries

- Keep the daemon focused on failover. It is not a recursive resolver, ad
  blocker, cache, policy router, or replacement for dnsmasq.
- dnsmasq remains the LAN-facing resolver and cache.
- The daemon initially accepts DNS over UDP and TCP and forwards to plain
  UDP/TCP upstreams, including local services such as `https-dns-proxy`.
- Native DoH, DoT, DoQ, DNSSEC validation, filtering, and split DNS are not MVP
  requirements.
- Do not silently change dnsmasq during package installation. Integration must
  be explicit, reversible, and preserve the previous UCI configuration.

## Required research workflow

Before changing behavior that depends on an external library, framework, API,
CLI, GitHub Action, or OpenWrt facility, fetch current documentation with the
Context7 CLI:

1. `npx ctx7@latest library "<official name>" "<full question>"`
2. Select the best official/high-reputation `/org/project` result.
3. `npx ctx7@latest docs <libraryId> "<full question>"`

Use no more than three Context7 CLI requests for one user question. If quota is
exhausted, report it and suggest `npx ctx7@latest login` or
`CONTEXT7_API_KEY`. Prefer official OpenWrt, LuCI, upstream project, and GitHub
documentation for facts not covered by Context7.

## Intended repository layout

```text
cmd/failsafe-dns-proxy/       daemon and CLI entry point
internal/config/              UCI-derived runtime configuration
internal/dnsserver/           UDP/TCP listeners and request handling
internal/failover/            strict-priority selection and state machine
internal/health/              active probes and recovery scheduling
internal/upstream/            transport interface and UDP/TCP implementations
internal/status/              atomic status snapshots/control socket
package/failsafe-dns-proxy/   OpenWrt package, procd service, UCI defaults
package/luci-app-failsafe-dns-proxy/ LuCI view, RPC/ACL and po/ translations
scripts/                      matrix generation, installer, release helpers
tests/integration/            fake upstream and end-to-end tests
docs/                         architecture and implementation plan
```

Do not create framework layers before they have at least two real consumers.

## Core behavior

Upstreams are ordered by ascending numeric priority. For each request:

1. Select the highest-priority upstream that is not `down`.
2. Forward with a per-attempt timeout.
3. On a transport error, timeout, malformed reply, `SERVFAIL`, or `REFUSED`,
   try the next eligible upstream while the total request deadline remains.
   Transport/protocol failures are global health evidence. A query-specific
   `SERVFAIL` or `REFUSED` triggers fallback but must not by itself mark the
   whole upstream down.
4. On a valid answer, return it unchanged and record success.
5. Never treat `NXDOMAIN` as an upstream failure.

The state machine must be deterministic and independently testable:

- `unknown`: not enough evidence yet;
- `healthy`: eligible and confirmed;
- `suspect`: recent failure, still eligible according to thresholds;
- `down`: skipped by normal traffic and checked by probes;
- `recovering`: probes are succeeding but the recovery threshold is not met;
- recovery requires consecutive successful probes;
- transitions use configurable failure/recovery thresholds;
- probes for down upstreams use bounded exponential backoff with jitter;
- a recovered upstream only becomes active if it has higher priority than the
  currently selected upstream.
- if every upstream is `down` or `recovering`, use an emergency half-open path
  that attempts them in priority order instead of returning immediate failure.

Use active and passive evidence. Active probes reduce the chance that a user
request pays the first timeout; passive failures catch outages between probes.
Avoid a global lock in the request path. Runtime selection should use immutable
configuration plus atomic or narrowly locked state.

Health state is intentionally in RAM for MVP. Do not write counters to flash on
every transition. After daemon restart, upstreams start as `unknown` and are
probed immediately.

## DNS correctness

- Support UDP and TCP listeners from the first release.
- Preserve transaction ID, question, flags, EDNS data, and valid response
  sections.
- Retry over TCP when an upstream UDP response is truncated.
- Validate that a response matches the request.
- Bound packet size, concurrent requests, per-attempt time, and total request
  time.
- A valid `NOERROR` or `NXDOMAIN` response proves transport health. Active-probe
  `SERVFAIL`/`REFUSED` responses count as probe failures. Passive query-specific
  `SERVFAIL`/`REFUSED` responses trigger fallback but need separate safeguards
  before changing global health.
- Upstream hostnames must not be bootstrapped through the same dnsmasq path in a
  way that creates a resolution loop. Require an IP or explicit bootstrap
  resolver until loop-safe hostname support is implemented.

## Configuration and OpenWrt integration

- The source of truth is `/etc/config/failsafe-dns-proxy`.
- Parse UCI through a small adapter; keep the failover core independent of UCI.
- Validate the complete configuration before replacing the running one.
- Reload atomically. Keep the old configuration if validation fails.
- Use `procd`, respawn limits, stdout/stderr logging, and reload triggers.
- Provide `failsafe-dns-proxy check-config` and
  `failsafe-dns-proxy status --json`.
- LuCI must edit UCI, display service/upstream state, and expose explicit
  apply/test actions. It must not contain failover logic.
- LuCI RPC permissions must follow least privilege. Prefer a narrow daemon
  control/status interface over unrestricted shell execution.
- If an enabled upstream points to a local `https-dns-proxy` listener, dnsmasq
  integration must reject or clearly warn when `https-dns-proxy` is still
  configured to manage dnsmasq itself. Require
  `https-dns-proxy.config.dnsmasq_config_update='-'` before making
  Failsafe DNS Proxy the dnsmasq upstream; otherwise both services can rewrite
  `/etc/config/dhcp` and create delayed local upstream timeouts.

## Go implementation rules

- Use a currently supported Go release that is available in the targeted
  OpenWrt build system.
- Prefer pure Go and `CGO_ENABLED=0`.
- Use `context.Context` for request deadlines and shutdown.
- Use `log/slog` or a small structured logging facade; never log every query by
  default.
- Return and wrap errors; do not panic for configuration, network, or packet
  errors.
- Close listeners and upstream transports during reload/shutdown.
- Add dependencies only when they materially reduce correctness risk.
- The initial DNS library should be `github.com/miekg/dns`.
- Do not import the complete AdGuard dnsproxy proxy engine for MVP. Re-evaluate
  its `upstream` package only when native encrypted transports are implemented.

## Testing requirements

Every behavior change needs tests at the lowest useful level.

- Unit-test all state transitions with a fake clock and deterministic random
  source.
- Unit-test priority selection, timeout budgeting, RCODE classification,
  recovery hysteresis, config validation, and concurrent updates.
- Integration-test UDP, TCP, UDP truncation/TCP retry, packet loss, delayed
  replies, malformed replies, outage, recovery, and reload.
- Run `go test -race ./...` on a non-embedded CI runner.
- Fuzz packet parsing and response validation where practical.
- Package workflows must smoke-test package contents and install scripts.

Initial resource targets:

- stripped daemon no larger than 8 MiB;
- idle RSS no larger than 12 MiB on a representative router;
- less than 2 ms local processing overhead at p95 with a healthy upstream;
- after an upstream is marked down, no failed-upstream timeout in the normal
  request path;
- clean shutdown/reload with no leaked goroutines under integration tests.

If a target is missed, record the measurement and reason instead of weakening
the test silently.

## Packaging and releases

Package names:

- `failsafe-dns-proxy`
- `luci-app-failsafe-dns-proxy`
- optional localization packages such as
  `luci-i18n-failsafe-dns-proxy-ru`

Use official OpenWrt SDKs and package Makefiles. Do not handcraft APK archives.
The release system must:

- build IPK for OpenWrt 24.10.x;
- build APK for OpenWrt 25.12.x and newer;
- deduplicate user-space daemon builds by OpenWrt package architecture where
  safe;
- build the LuCI package as architecture `all`;
- provide a manual workflow for one exact
  version/target/subtarget combination;
- provide checksums and a machine-readable release manifest;
- never ignore compilation errors with `|| true`;
- pin GitHub Actions by commit SHA before the first stable release.

The installer must detect OpenWrt version, target, subtarget, package
architecture, and package manager through `ubus` with documented fallbacks. It
must verify checksums before installation and fail without modifying dnsmasq if
no exact compatible artifact exists.

## Change discipline

- Keep commits and patches scoped.
- Preserve unrelated user changes.
- Update README and plan when architecture, compatibility, config schema, or
  release behavior changes.
- Do not claim support for a target that CI has not built or tested.
- Do not merge placeholder security, signature, or error-handling code.
