# Failsafe DNS Proxy for OpenWrt

[![CI](https://github.com/nikitadybov/owrt-failsafe-dns-proxy-go/actions/workflows/ci.yml/badge.svg)](https://github.com/nikitadybov/owrt-failsafe-dns-proxy-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/daemon-MIT-blue.svg)](LICENSE)

Небольшой DNS failover daemon для OpenWrt. Он использует upstream-серверы
строго по приоритету, запоминает временно отказавшие серверы и автоматически
возвращается на основной upstream только после подтверждённого восстановления.

Проект не заменяет dnsmasq. dnsmasq остаётся LAN-facing DNS-сервером, кэшем и
источником локальных имён:

```text
клиенты LAN
    |
    v
dnsmasq
    |
    v
failsafe-dns-proxy 127.0.0.1:5359
    |-- priority 10 -> локальный https-dns-proxy 127.0.0.1:5054
    `-- priority 20 -> резервный DNS 77.88.8.8:53
```

## Возможности

- UDP и TCP для клиентов и обычных DNS upstream;
- повтор запроса по TCP после truncated UDP-ответа;
- строгий числовой приоритет upstream;
- состояния `unknown`, `healthy`, `suspect`, `down` и `recovering`;
- пассивное обнаружение отказов и фоновые active probes;
- ограниченные таймауты одной попытки и всего клиентского запроса;
- recovery hysteresis и автоматический failback;
- emergency half-open, если все upstream считаются недоступными;
- проверка DNS transaction ID, question и структуры ответа;
- atomic reload с сохранением рабочей конфигурации при ошибке;
- UCI, procd, JSON status, LuCI и русская локализация;
- явная обратимая интеграция с dnsmasq;
- IPK для OpenWrt 24.10 и APK для OpenWrt 25.12;
- exact-match installer с проверкой SHA-256.

## Статус проекта

Версия пакета daemon: `0.1.0-r12`.

Основной сценарий проверен на Xiaomi Redmi Router AX6S с OpenWrt 24.10.7:
установка, LuCI, отказ основного upstream, переход на резервный, recovery,
failback, reload, dnsmasq rollback и package lifecycle. Продолжается
многодневный soak test.

Сборки для других платформ считаются проверенными компиляцией и package smoke
test, но не реальным устройством. Точный журнал готовности находится в
[docs/PLAN.md](docs/PLAN.md).

## Совместимость

| OpenWrt | Формат | Менеджер | Статус |
| --- | --- | --- | --- |
| 24.10.7 | IPK | `opkg` | default release build для `mediatek/mt7622`, протестировано на AX6S |
| 25.12.4 | APK | `apk` | default release build для `mediatek/mt7622`, package smoke test |

Release matrix описана в
[build/release-targets.json](build/release-targets.json). Installer принимает
только точное совпадение version, target, subtarget, package architecture и
package manager.

Основной Release намеренно ограничен роутером Xiaomi Redmi Router AX6S
(`mediatek/mt7622`) и двумя точными версиями OpenWrt: `24.10.7` и `25.12.4`.
Для любой другой версии или платформы пользователь собирает собственный
artifact в GitHub fork через workflow **Build one OpenWrt target**.

## Установка

### Из GitHub Release

После публикации release скачайте `install.sh` на роутер и запустите его с URL
manifest и каталога этого же release:

```sh
RELEASE=v0.1.0
BASE="https://github.com/nikitadybov/owrt-failsafe-dns-proxy-go/releases/download/$RELEASE"

uclient-fetch -O /tmp/install-failsafe-dns-proxy.sh "$BASE/install.sh"
chmod +x /tmp/install-failsafe-dns-proxy.sh

/tmp/install-failsafe-dns-proxy.sh \
  --manifest "$BASE/manifest.json" \
  --base-url "$BASE"
```

По умолчанию устанавливаются daemon, LuCI и русская локализация. Варианты:

```sh
# Без русской локализации
/tmp/install-failsafe-dns-proxy.sh \
  --manifest "$BASE/manifest.json" \
  --base-url "$BASE" \
  --no-russian

# Только daemon
/tmp/install-failsafe-dns-proxy.sh \
  --manifest "$BASE/manifest.json" \
  --base-url "$BASE" \
  --daemon-only
```

Installer не изменяет dnsmasq без флага `--configure-dnsmasq`. Для первой
установки безопаснее сначала настроить и проверить proxy, а интеграцию включить
отдельно.

### Из локально собранных пакетов

Скопируйте на роутер `manifest.json`, соответствующие пакеты и installer:

```sh
./install.sh \
  --manifest ./manifest.json \
  --source-dir .
```

Пакеты можно установить вручную:

```sh
# OpenWrt 24.10
opkg install ./failsafe-dns-proxy_*.ipk
opkg install ./luci-app-failsafe-dns-proxy_*.ipk
opkg install ./luci-i18n-failsafe-dns-proxy-ru_*.ipk

# OpenWrt 25.12
apk --allow-untrusted add ./failsafe-dns-proxy-*.apk
apk --allow-untrusted add ./luci-app-failsafe-dns-proxy-*.apk
apk --allow-untrusted add ./luci-i18n-failsafe-dns-proxy-ru-*.apk
```

Установка пакета не включает сервис и не меняет dnsmasq.

### Если вашей платформы нет в Release

Не устанавливайте пакет от похожего роутера только потому, что процессор имеет
ту же архитектуру. Installer специально требует точное совпадение OpenWrt
version, target, subtarget и package architecture.

Сначала определите параметры своего роутера:

```sh
ubus call system board
```

Нужны поля:

```json
{
  "release": {
    "version": "25.12.4",
    "target": "mediatek/filogic"
  }
}
```

В этом примере параметры сборки:

```text
openwrt_version: 25.12.4
target: mediatek
subtarget: filogic
```

Package architecture можно посмотреть отдельно:

```sh
# OpenWrt 24.10
opkg print-architecture

# OpenWrt 25.12
apk --print-arch
```

#### Автоматическая сборка через GitHub fork

1. Войдите в GitHub и откройте репозиторий проекта.
2. Нажмите **Fork → Create fork**.
3. В своём fork откройте вкладку **Actions**.
4. Если GitHub показывает предупреждение, нажмите
   **I understand my workflows, go ahead and enable them**.
5. Слева выберите **Build one OpenWrt target**.
6. Нажмите **Run workflow**.
7. Укажите точные значения:
   - `openwrt_version`, например `25.12.4`;
   - `target`, например `mediatek`;
   - `subtarget`, например `filogic`;
   - `publish_artifact` оставьте включённым.
8. Нажмите зелёную кнопку **Run workflow** и дождитесь завершения job.
9. Откройте завершённый запуск и скачайте artifact
   `openwrt-VERSION-TARGET-SUBTARGET` внизу страницы.

Workflow автоматически:

- найдёт официальный SDK для указанной комбинации;
- скачает официальные `sha256sums` и проверит SDK;
- определит IPK или APK по версии OpenWrt;
- соберёт daemon, LuCI и русскую локализацию;
- проверит содержимое пакетов;
- добавит `SHA256SUMS` и `build-metadata.json`;
- завершится с ошибкой, если официального SDK или поддерживаемого package
  format нет.

Сейчас generic builder поддерживает официальные серии:

- OpenWrt `24.10.x` — IPK и `opkg`;
- OpenWrt `25.12.x` — APK и `apk`.

Произвольные старые версии, development snapshots, OpenWrt forks и будущие
серии автоматически не поддерживаются. Для них может потребоваться изменение
package recipe или build script.

#### Установка пользовательской сборки

Распакуйте скачанный GitHub artifact на компьютере и проверьте checksums:

```sh
sha256sum -c SHA256SUMS
```

Скопируйте нужные пакеты на роутер, например:

```sh
scp failsafe-dns-proxy* luci-app-failsafe-dns-proxy* \
  luci-i18n-failsafe-dns-proxy-ru* root@192.168.1.1:/tmp/
```

Доступны три пакета:

| Пакет | Обязателен | Назначение |
| --- | --- | --- |
| `failsafe-dns-proxy` | да | daemon, CLI, UCI, procd и dnsmasq helpers |
| `luci-app-failsafe-dns-proxy` | нет | настройка и мониторинг через LuCI |
| `luci-i18n-failsafe-dns-proxy-ru` | нет | русский перевод LuCI |

Для минимальной установки достаточно daemon:

```sh
# OpenWrt 24.10
opkg install /tmp/failsafe-dns-proxy_*.ipk

# OpenWrt 25.12
apk --allow-untrusted add /tmp/failsafe-dns-proxy-*.apk
```

Для полной установки:

```sh
# OpenWrt 24.10
opkg install /tmp/failsafe-dns-proxy_*.ipk
opkg install /tmp/luci-app-failsafe-dns-proxy_*.ipk
opkg install /tmp/luci-i18n-failsafe-dns-proxy-ru_*.ipk

# OpenWrt 25.12
apk --allow-untrusted add /tmp/failsafe-dns-proxy-*.apk
apk --allow-untrusted add /tmp/luci-app-failsafe-dns-proxy-*.apk
apk --allow-untrusted add /tmp/luci-i18n-failsafe-dns-proxy-ru-*.apk
```

`Build one OpenWrt target` создаёт build artifact, а не официальный Release
проекта. Поэтому для такой сборки используется ручная установка пакетов.
Пользователь самостоятельно отвечает за проверку на своём устройстве. Успешная
компиляция подтверждает совместимость SDK и package format, но не работу
конкретного роутера.

После установки выполните шаги из раздела [Быстрый запуск](#быстрый-запуск).

## Быстрый запуск

1. Откройте LuCI → **Services → Failsafe DNS Proxy** или измените
   `/etc/config/failsafe-dns-proxy`.
2. Замените примерные upstream своими адресами.
3. Включите daemon и проверьте конфигурацию:

```sh
uci set failsafe-dns-proxy.main.enabled='1'
uci commit failsafe-dns-proxy

failsafe-dns-proxy check-config
/etc/init.d/failsafe-dns-proxy enable
/etc/init.d/failsafe-dns-proxy start
failsafe-dns-proxy self-test
failsafe-dns-proxy status
```

1. Только после успешного self-test направьте dnsmasq на proxy:

```sh
failsafe-dns-proxy-dnsmasq dry-run
failsafe-dns-proxy-dnsmasq enable
failsafe-dns-proxy-dnsmasq status
```

Команда `enable` сохраняет предыдущую UCI-конфигурацию dnsmasq, применяет
`noresolv=1` и `server=127.0.0.1#5359`, проверяет DNS и выполняет rollback при
ошибке.

## Текущая конфигурация

### Настройки, поставляемые пакетом

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

`enabled` намеренно равен `0`: пакет не должен запускать непроверенную
конфигурацию автоматически.

На тестовом AX6S сохранена эта же схема, но daemon включён:
`failsafe-dns-proxy.main.enabled=1`. Основной upstream — локальный
`https-dns-proxy` на `127.0.0.1:5054`, резервный — `77.88.8.8:53`.

### Значение параметров

| Параметр | Сейчас | Назначение |
| --- | ---: | --- |
| `listen_addr` | `127.0.0.1` | адрес listener; loopback безопасен для схемы с dnsmasq |
| `listen_port` | `5359` | UDP и TCP порт proxy |
| `attempt_timeout_ms` | `700` | максимум одной попытки к upstream |
| `request_timeout_ms` | `2000` | общий бюджет запроса со всеми fallback |
| `health_interval_s` | `5` | базовый интервал active probes |
| `fail_threshold` | `2` | число подтверждённых ошибок до состояния `down` |
| `recover_threshold` | `2` | число успешных probes до восстановления |
| `max_concurrent` | `128` | общий предел одновременно обрабатываемых запросов |
| `probe` | два DNS-вопроса | вопросы для фоновой проверки транспорта |
| `priority` | `10`, `20` | меньшее число означает более высокий приоритет |
| `protocol` | `udp` | plain DNS transport; допустимы `udp` и `tcp` |

`request_timeout_ms` должен быть не меньше `attempt_timeout_ms`. Если
upstream-серверов несколько, общий бюджет должен оставлять время хотя бы для
одной резервной попытки.

## Как изменить настройки

### Через LuCI

Откройте **Services → Failsafe DNS Proxy**. Интерфейс позволяет:

- включить сервис и изменить listener;
- добавить, отключить или удалить upstream;
- изменить приоритет, протокол, IP и порт;
- настроить таймауты, thresholds и probe-вопросы;
- проверить конфигурацию и отдельный upstream;
- увидеть активный upstream, health state и последние ошибки;
- включить или отключить интеграцию с dnsmasq.

### Через UCI

Пример замены резервного DNS:

```sh
uci set failsafe-dns-proxy.public_fallback.address='1.1.1.1'
uci set failsafe-dns-proxy.public_fallback.port='53'
uci commit failsafe-dns-proxy

failsafe-dns-proxy check-config
/etc/init.d/failsafe-dns-proxy reload
```

Добавление ещё одного upstream:

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

Имена upstream и значения `priority` должны быть уникальны. Адрес upstream
пока обязан быть IP-адресом. Hostname может создать bootstrap loop, если
dnsmasq уже направлен на этот proxy.

Reload применяется атомарно. Ошибочная конфигурация отклоняется, а daemon
продолжает работать со старой. Изменение `listen_addr`, `listen_port` или
`status_socket` требует restart:

```sh
/etc/init.d/failsafe-dns-proxy restart
```

## Эксплуатация

Основные команды:

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

Daemon не логирует каждый DNS-запрос. В журнал попадают operational events:
отказ, переход в recovery, восстановление, failover/failback, rejected reload
и emergency mode.

### Многодневный soak test

```sh
failsafe-dns-proxy-soak start 168 60
failsafe-dns-proxy-soak status
failsafe-dns-proxy-soak report
failsafe-dns-proxy-soak stop
```

Первый аргумент — длительность в часах, второй — интервал измерений в секундах.
Monitor записывает RSS, goroutines, file descriptors, heap, active upstream и
результат self-test в `/tmp/failsafe-dns-proxy-soak`.

### Отключение dnsmasq-интеграции

```sh
failsafe-dns-proxy-dnsmasq disable
```

Команда восстанавливает сохранённые параметры. Если UCI package `dhcp` был
изменён после включения интеграции, автоматическое восстановление будет
отклонено, чтобы не затереть более новые настройки.

## Как работает failover

Для каждого запроса выбирается доступный upstream с минимальным `priority`.
Timeout, transport error, malformed response и mismatch считаются глобальными
health evidence и позволяют попробовать следующий upstream. Пользовательский
`SERVFAIL` или `REFUSED` запускает fallback, но одной такой ошибки недостаточно
для глобального отключения сервера. `NOERROR` и `NXDOMAIN` являются успешными
transport-ответами.

После перехода upstream в `down` обычные запросы больше не платят его timeout.
Фоновые probes проверяют его с bounded exponential backoff. Восстановление
требует последовательных успехов. Более приоритетный upstream автоматически
возвращается в работу после recovery threshold.

Health state хранится в RAM. После restart все upstream начинают как
`unknown` и немедленно проверяются.

## Ограничения

- Native DoH, DoT и DoQ пока не реализованы.
- Upstream должен быть IP-адресом; loop-safe hostname bootstrap отсутствует.
- Нет DNS cache, filtering, DNSSEC validation, split DNS и load balancing.
- dnsmasq остаётся обязательным для LAN cache и локальных имён.
- APK-сборка не означает аппаратную проверку на каждом target.
- Поддерживается официальный OpenWrt; forks требуют отдельной валидации.

## Разработка

### Требования

- Go версии из `go.mod`;
- GNU Make;
- Node.js для проверки LuCI JavaScript и Markdown;
- ShellCheck;
- Linux x86_64, `curl`, `zstd` и официальный OpenWrt SDK для package builds.

### Проверки

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

### Локальная сборка daemon

```sh
make build
make cross-build
make check-config
```

### Сборка OpenWrt-пакетов

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

### GitHub Actions

- `ci.yml` — Go tests, race, fuzz smoke, static checks, IPK и APK smoke builds;
- `build-one.yml` — ручная сборка exact version/target/subtarget;
- `build-openwrt.yml` — reusable SDK build;
- `release.yml` — две default-сборки `mediatek/mt7622` для OpenWrt 24.10.7 и
  25.12.4, manifest, checksums и GitHub Release.

Release tag должен совпадать с `PKG_VERSION`, например `v0.1.0`. Actions
закреплены по commit SHA. До первого успешного запуска workflow на GitHub
release pipeline считается реализованным локально, но не подтверждённым
публичной публикацией.

`build-one.yml` доступен пользователям через fork и предназначен именно для
всех остальных версий и платформ. Пользовательская сборка не добавляется в
основной Release проекта автоматически.

### Структура репозитория

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
уровне и обновлением [docs/PLAN.md](docs/PLAN.md).

## Диагностика

### Сервис не запускается

```sh
failsafe-dns-proxy check-config
logread -e failsafe-dns-proxy
```

Проверьте `enabled=1`, уникальные priority, IP-адреса upstream и соотношение
таймаутов.

### Status socket недоступен

Проверьте, что daemon запущен и `status_socket` совпадает с CLI:

```sh
failsafe-dns-proxy status \
  --socket /var/run/failsafe-dns-proxy.sock
```

### После настройки клиенты обходят proxy

Проверьте dnsmasq:

```sh
uci show dhcp.@dnsmasq[0].noresolv
uci show dhcp.@dnsmasq[0].server
failsafe-dns-proxy-dnsmasq status
```

Для строгой схемы нужен `noresolv=1`; иначе dnsmasq может использовать DNS из
DHCP параллельно.

### Fallback слишком медленный

Уменьшайте `attempt_timeout_ms` осторожно и только после измерений. Слишком
малое значение создаёт ложные отказы на загруженном роутере или медленном
канале.

## Безопасность и сообщения об ошибках

Не публикуйте конфигурации с приватными адресами, credentials или полными
сетевыми логами. Security issue лучше сообщать приватно владельцу репозитория,
не раскрывая exploit в публичном issue до исправления.

## Лицензии

Daemon и основная часть репозитория распространяются по [MIT](LICENSE).
LuCI package использует Apache-2.0, как указано в его package metadata.
