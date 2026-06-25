# Contributing

**English** | [Русский](../ru/contributing.md)

The project is maintained in spare time and grows through real use on routers.
Contributions are welcome — code, documentation, packaging, translations, bug
reports, and hardware validation all help.

## How you can help

- **Improve the daemon** — failover logic, health checks, performance, tests.
- **Extend LuCI and docs** — clearer UI, better guides, new translations.
- **Fix packaging and CI** — OpenWrt recipes, installer, GitHub Actions.
- **Report issues** — reproducible steps, logs, and OpenWrt version details.
- **Maintain the project** — review pull requests, triage issues, keep
  dependencies and workflows current.

Before a non-trivial change, read [PLAN.md](../PLAN.md) and
[AGENTS.md](../../AGENTS.md). Behavior changes should include tests at the
lowest useful level and an update to the plan when scope or status changes.

Development setup and checks: [Development](development.md).

## Add a tested device

Official releases currently cover a small matrix. Many combinations are built
only through **Build one OpenWrt target** and validated by users on real
hardware. If you ran the daemon successfully on your router, please share that
result so others know the combination is safe to try.

### What to test

At minimum, confirm on your device:

- package install and service start;
- primary upstream failure and fallback to backup;
- recovery and failback to the primary upstream;
- config reload and, if used, dnsmasq integration enable/disable.

Optional but valuable: LuCI, upgrade/remove lifecycle, multi-day soak.

### What to include in a pull request or issue

| Field | Example |
| --- | --- |
| Device | Xiaomi Redmi Router AX6S |
| OpenWrt version | 24.10.7 |
| Target / subtarget | `mediatek` / `mt7622` |
| Package architecture | `aarch64_cortex-a53` |
| Package format | IPK / `opkg` or APK / `apk` |
| How installed | release, fork build, local SDK build |
| What was verified | install, failover, failback, reload, … |

Attach relevant command output if possible (`ubus call system board`,
`failsafe-dns-proxy status --json`, short log excerpts). Do not publish private
networks, credentials, or full query logs.

### Files to update

For a **documented hardware test** (without adding a default release target):

- [docs/en/compatibility.md](compatibility.md) and
  [docs/ru/compatibility.md](../ru/compatibility.md) — add a row or note under
  community-tested devices;
- [docs/en/project-status.md](project-status.md) and
  [docs/ru/project-status.md](../ru/project-status.md) — if the result
  materially changes the readiness picture.

For a **new default release target** (maintainer-reviewed, CI-built):

- [build/release-targets.json](../../build/release-targets.json) — set
  `tested_on_hardware` honestly and keep version/target/subtarget exact;
- compatibility and project-status docs;
- [PLAN.md](../PLAN.md) — release and verification status.

Do not mark a target as tested on hardware unless you actually ran the
scenarios above on that device. Compile-only or smoke-build results belong in
compatibility notes, not as hardware-verified claims.

## Pull requests

1. Fork the repository and create a focused branch.
2. Run local checks: `make lint` and `make ci` when Go or shell code changes.
3. Describe what changed, why, and how you tested it.
4. Keep unrelated edits out of the same pull request.

Questions and work-in-progress are fine as draft pull requests or issues.

## Code of conduct

Be constructive and precise. Security issues should be reported privately —
see [Security](security.md).

[← Documentation index](index.md)
