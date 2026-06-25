# Compatibility

**English** | [Русский](../ru/compatibility.md)

| OpenWrt | Format | Package manager | Status |
| --- | --- | --- | --- |
| 24.10.7 | IPK | `opkg` | default release build for `mediatek/mt7622`, tested on AX6S |
| 25.12.4 | APK | `apk` | default release build for `mediatek/mt7622`, package smoke test |

The release matrix is defined in
[build/release-targets.json](../../build/release-targets.json). The installer
accepts only an exact match of version, target, subtarget, package
architecture, and package manager.

The main release intentionally targets the Xiaomi Redmi Router AX6S
(`mediatek/mt7622`) and two exact OpenWrt versions: `24.10.7` and `25.12.4`.
For any other version or platform, build your own artifact in a GitHub fork
using the **Build one OpenWrt target** workflow. See
[Build for your platform](../../README.md#build-for-your-platform).

## Community-tested devices

Hardware verification beyond the default release matrix is tracked here. To add
your device, follow [Contributing → Add a tested device](contributing.md#add-a-tested-device).

| Device | OpenWrt | Target | Verified by project | Notes |
| --- | --- | --- | --- | --- |
| Xiaomi Redmi Router AX6S | 24.10.7 | `mediatek/mt7622` | yes | primary reference device; full MVP scenario |

If your combination is not listed, a successful fork build plus the tests
described in the contributing guide is enough to propose an entry.

[← Documentation index](index.md)
