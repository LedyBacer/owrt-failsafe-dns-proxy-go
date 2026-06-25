# Failover behavior

**English** | [Русский](../ru/failover.md)

For each request the daemon selects the available upstream with the lowest
`priority`. Timeout, transport error, malformed response, and mismatch count as
global health evidence and allow trying the next upstream. A query-specific
`SERVFAIL` or `REFUSED` triggers fallback, but a single such response is not
enough to mark the whole server down globally. `NOERROR` and `NXDOMAIN` are
successful transport responses.

After an upstream enters `down`, normal requests no longer pay its timeout.
Background probes check it with bounded exponential backoff. Recovery requires
consecutive successes. A higher-priority upstream returns to service
automatically after the recovery threshold.

Health state is stored in RAM. After restart all upstreams start as `unknown`
and are probed immediately.

[← Documentation index](index.md)
