# Разработка

[English](../en/development.md) | **Русский**

## Требования

- Go версии из `go.mod`;
- GNU Make;
- Node.js для проверки LuCI JavaScript и Markdown;
- ShellCheck;
- Linux x86_64, `curl`, `zstd` и официальный OpenWrt SDK для package builds.

## Проверки

```sh
make test
make race
make vet
make lint
make ci
```

`make lint` проверяет Go formatting, `go vet`, shell scripts, LuCI JavaScript,
Markdown, JSON, Python syntax и GitHub Actions. `make ci` дополнительно запускает
обычные и race tests.

## Локальная сборка daemon

```sh
make build
make cross-build
make check-config
```

## Сборка OpenWrt-пакетов

Generic builder получает точный официальный SDK из OpenWrt downloads,
проверяет его SHA-256 и собирает daemon, LuCI и русскую локализацию:

```sh
./scripts/build-openwrt.sh 24.10.7 mediatek mt7622 ./dist
./scripts/build-openwrt.sh 25.12.4 mediatek mt7622 ./dist
```

Для AX6S/OpenWrt 24.10.7 есть короткий wrapper:

```sh
./scripts/build-openwrt-24.10.7-ax6s.sh
```

Официальные SDK запускаются на Linux x86_64. Результат содержит пакеты,
`SHA256SUMS` и `build-metadata.json`.

Для любой поддерживаемой официальной комбинации синтаксис одинаков:

```sh
./scripts/build-openwrt.sh OPENWRT_VERSION TARGET SUBTARGET ./dist
```

Значения `TARGET` и `SUBTARGET` берутся из поля `release.target` команды
`ubus call system board`. Например, `mediatek/filogic` преобразуется в:

```sh
./scripts/build-openwrt.sh 25.12.4 mediatek filogic ./dist
```

## GitHub Actions

- `ci.yml` — Go tests, race, fuzz smoke, static checks, IPK и APK smoke builds;
- `build-one.yml` — ручная сборка exact version/target/subtarget;
- `build-openwrt.yml` — reusable SDK build;
- `release.yml` — две default-сборки `mediatek/mt7622` для OpenWrt 24.10.7 и
  25.12.4, manifest, checksums и GitHub Release.

Release tag должен совпадать с `PKG_VERSION`, например `v0.2.0`. Actions
закреплены по commit SHA. До первого успешного запуска workflow на GitHub
release pipeline считается реализованным локально, но не подтверждённым
публичной публикацией.

`build-one.yml` доступен пользователям через fork и предназначен именно для
всех остальных версий и платформ. Пользовательская сборка не добавляется в
основной Release проекта автоматически.

## Структура репозитория

```text
cmd/failsafe-dns-proxy/              daemon и CLI
internal/                            config, DNS, failover, health, status
package/failsafe-dns-proxy/          daemon package и OpenWrt integration
package/luci-app-failsafe-dns-proxy/ LuCI, RPC/ACL и ru localization
scripts/                             build, release, installer, soak helpers
tests/                               integration и shell tests
docs/PLAN.md                         roadmap и status ledger
```

Изменения поведения должны сопровождаться тестом на минимальном полезном
уровне и обновлением [PLAN.md](../PLAN.md).

[← Оглавление](index.md)
