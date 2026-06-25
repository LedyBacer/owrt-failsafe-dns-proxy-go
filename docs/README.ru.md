# Failsafe DNS Proxy для OpenWrt

[English](../README.md) | **Русский**

[![CI](https://github.com/LedyBacer/owrt-failsafe-dns-proxy-go/actions/workflows/ci.yml/badge.svg)](https://github.com/LedyBacer/owrt-failsafe-dns-proxy-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/daemon-MIT-blue.svg)](../LICENSE)

**Автоматический failover DNS upstream на OpenWrt, когда основной резолвер
перестаёт работать.**

На домашнем роутере часто нужен основной DNS — например, локальный
`https-dns-proxy` с шифрованием — и простой резерв: DNS провайдера или
публичный сервер. dnsmasq умеет указывать только один upstream-адрес и не
следит за его состоянием. Если основной сервис завис, упал или потерял
связь, DNS в LAN перестаёт работать, пока вы вручную не смените настройки.

Failsafe DNS Proxy стоит между dnsmasq и вашими upstream-резолверами. dnsmasq
по-прежнему обслуживает клиентов, кэш и локальные имена; daemon — единственный
upstream для dnsmasq и берёт на себя остальное:

- направляет запросы upstream-ам строго по приоритету;
- быстро пропускает отказавший upstream, а не ждёт его на каждом запросе;
- переключается на следующий резолвер в пределах заданных таймаутов;
- проверяет упавшие upstream-ы в фоне и возвращается на основной только после
  подтверждённого восстановления.

Типичный сценарий: **основной** `127.0.0.1:5054` (`https-dns-proxy`) →
**резерв** `77.88.8.8:53`. Проект не заменяет dnsmasq, не блокирует рекламу,
не кэширует DNS и не является DoH/DoT-клиентом — это только небольшой слой
failover для выбора upstream.

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

Полная документация: [English](en/index.md) | [Русский](ru/index.md) —
начните с [обзора](ru/overview.md).

## Установка

### Из GitHub Release

После публикации release скачайте `install.sh` на роутер и запустите его с URL
manifest и каталога этого же release:

```sh
RELEASE=v0.1.0
BASE="https://github.com/LedyBacer/owrt-failsafe-dns-proxy-go/releases/download/$RELEASE"

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

После установки перейдите к [быстрому запуску](ru/quick-start.md).

## Сборка под свою платформу

Официальные release покрывают ограниченный набор версий OpenWrt и платформ. Не
устанавливайте пакет от похожего роутера только потому, что процессор имеет
ту же архитектуру. Installer требует точное совпадение OpenWrt version, target,
subtarget и package architecture.

Определите параметры роутера:

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

Package architecture:

```sh
# OpenWrt 24.10
opkg print-architecture

# OpenWrt 25.12
apk --print-arch
```

### Автоматическая сборка через GitHub fork

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

Поддерживаемые официальные серии:

- OpenWrt `24.10.x` — IPK и `opkg`;
- OpenWrt `25.12.x` — APK и `apk`.

Произвольные старые версии, development snapshots, OpenWrt forks и будущие
серии автоматически не поддерживаются.

### Установка пользовательской сборки

Распакуйте скачанный GitHub artifact на компьютере и проверьте checksums:

```sh
sha256sum -c SHA256SUMS
```

Скопируйте пакеты на роутер:

```sh
scp failsafe-dns-proxy* luci-app-failsafe-dns-proxy* \
  luci-i18n-failsafe-dns-proxy-ru* root@192.168.1.1:/tmp/
```

| Пакет | Обязателен | Назначение |
| --- | --- | --- |
| `failsafe-dns-proxy` | да | daemon, CLI, UCI, procd и dnsmasq helpers |
| `luci-app-failsafe-dns-proxy` | нет | настройка и мониторинг через LuCI |
| `luci-i18n-failsafe-dns-proxy-ru` | нет | русский перевод LuCI |

Минимальная установка:

```sh
# OpenWrt 24.10
opkg install /tmp/failsafe-dns-proxy_*.ipk

# OpenWrt 25.12
apk --allow-untrusted add /tmp/failsafe-dns-proxy-*.apk
```

Полная установка:

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

**Build one OpenWrt target** создаёт build artifact, а не официальный Release
проекта. Используйте ручную установку пакетов и проверку на своём устройстве.
Успешная компиляция подтверждает совместимость SDK и package format, но не
работу конкретного роутера.

После установки перейдите к [быстрому запуску](ru/quick-start.md).

## Участие в проекте

Вклад приветствуется: код, документация, packaging, переводы, багрепорты и
проверка на реальном железе. Проект остаётся полезным, когда его дополняют,
дорабатывают, поддерживают и делятся результатами с роутеров вне default
release matrix.

Если daemon успешно отработал на вашем устройстве, откройте pull request или
issue с моделью роутера, версией OpenWrt, target/subtarget и тем, что
проверили (install, failover, failback, reload). Так другим проще выбрать
проверенную комбинацию.

Подробности: [руководство для contributors](ru/contributing.md).
