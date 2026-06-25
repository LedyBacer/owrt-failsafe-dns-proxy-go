# OpenWrt Failsafe DNS Proxy

Лёгкий DNS-прокси с запоминанием отказавших upstream-серверов, строгим
приоритетом и автоматическим возвратом на основной DNS после восстановления.

Проект рассчитан на схему:

```text
клиенты LAN
    |
    v
dnsmasq (кэш и локальные имена)
    |
    v
failsafe-dns-proxy :5359
    |-- priority 10 -> https-dns-proxy 127.0.0.1:5054
    |-- priority 20 -> DNS провайдера
    `-- priority 30 -> дополнительный резервный DNS
```

Если основной upstream перестал отвечать, первый обнаруживший проблему запрос
может дождаться ограниченного таймаута. Следующие запросы не отправляются на
запомненный неработающий сервер. Фоновые проверки продолжаются, и после
нескольких успешных ответов более приоритетный upstream снова становится
активным.

> Статус: P0/P1 engineering завершён и проверен на Xiaomi Redmi Router AX6S.
> На роутере запущен автономный 7-дневный soak-monitor.

## Что уже реализовано

- UDP и TCP listeners;
- plain UDP/TCP upstreams;
- повтор через TCP после truncated UDP response;
- strict-priority failover с состояниями `unknown`, `healthy`, `suspect`,
  `down`, `recovering`;
- failure/recovery thresholds, emergency half-open и active probes;
- ограниченные attempt/request timeout и число одновременных запросов;
- проверка transaction ID и DNS question;
- корректная обработка `NXDOMAIN`, `SERVFAIL` и `REFUSED`;
- UCI-конфигурация, `check-config`, JSON status через Unix socket;
- procd service и OpenWrt package recipe;
- atomic SIGHUP reload без смены listener и с сохранением старой рабочей
  конфигурации при ошибке;
- явная обратимая dnsmasq-интеграция с backup, dry-run, DNS verification и
  rollback;
- ограничение параллельных попыток к одному зависшему upstream;
- runtime-метрики goroutine/FD/heap и автономный soak-monitor;
- LuCI: runtime status, UCI configuration, service controls, upstream probe и
  журнал переключений;
- русская локализация и отдельный least-privilege rpcd API;
- operational logs только для отказов, восстановления, failover/failback и
  emergency half-open;
- exact-match installer с проверкой SHA-256;
- GitHub Actions для unit/race/vet/fuzz/static/package smoke;
- unit/integration/race/fuzz tests;
- проверенный IPK build официальным SDK для Xiaomi Redmi Router AX6S на
  OpenWrt 24.10.7.

## Локальная проверка

Требуется Go 1.23.x — это версия Go в packages feed OpenWrt 24.10.

```sh
make test
make race
make vet
make cross-build
make check-config
```

## Сборка IPK для AX6S

Официальный SDK OpenWrt 24.10.7 поставляется для Linux x86_64. На таком host
или CI runner:

```sh
./scripts/build-openwrt-24.10.7-ax6s.sh
```

Скрипт проверяет checksum SDK и собирает IPK для:

```text
version: 24.10.7
target: mediatek/mt7622
package architecture: aarch64_cortex-a53
```

Результат копируется в `dist/`. Установка пакета не меняет dnsmasq и не
включает сервис автоматически. Перед запуском нужно отредактировать
`/etc/config/failsafe-dns-proxy`, установить `option enabled '1'`, затем:

```sh
/etc/init.d/failsafe-dns-proxy enable
/etc/init.d/failsafe-dns-proxy start
failsafe-dns-proxy status --json
```

Проверенные артефакты:

```text
failsafe-dns-proxy_0.1.0-r12_aarch64_cortex-a53.ipk
SHA-256: 0d07cc590ddf5a410b052a31ecff54ce054f54e6a9cd780a1d5c2af08ebb86b4

luci-app-failsafe-dns-proxy_0.1.0-r3_all.ipk
SHA-256: f35cd2dc3d36f865d98df78ec51caa7ae1100cdc8b37b3ad750469a0698ea36d

luci-i18n-failsafe-dns-proxy-ru_0.260625.55965_all.ipk
SHA-256: 8f71e116dd76e7cd72f8408f0e6b1b0e410e4328b1ece68ed30affb713501a84
```

Бинарник внутри пакета — stripped, statically linked ARM64 ELF размером около
4.9 MiB. Размер IPK — около 2.0 MiB.

## Зачем нужен отдельный проект

Сценарий проекта отличается от обычной балансировки DNS. Во время временной
сетевой фильтрации публичный или зашифрованный DNS может полностью перестать
отвечать, а DNS интернет-провайдера продолжает работать. Последовательный
fallback без памяти заставляет каждый новый запрос снова ждать таймаут
недоступного сервера.

Ценность этого проекта — не сам fallback, а сочетание:

- строгого порядка приоритетов;
- активных health checks;
- пассивного обнаружения ошибки на пользовательском запросе;
- запоминания состояния между запросами;
- hysteresis: несколько ошибок для отключения и несколько успехов для
  восстановления;
- автоматического failback на более приоритетный DNS;
- простой установки и настройки в OpenWrt/LuCI.

## Аналоги и результат исследования

Существующие проекты подтверждают техническую реализуемость идеи:

- [SmartDNS](https://github.com/pymumu/smartdns) — зрелый DNS-сервер с
  несколькими upstream, fallback, выбором быстрых ответов и OpenWrt/LuCI
  пакетами. Он решает существенно более широкую задачу и ориентирован прежде
  всего на выбор ответа/адреса, а не на минимальный strict-priority failover.
- [mosdns](https://github.com/IrineSistiana/mosdns) — программируемый DNS
  forwarder на Go с fallback-плагином и OpenWrt-сценариями. Нужное поведение
  можно собрать конфигурацией, но итог сложнее для обычного пользователя и не
  даёт специализированного интерфейса состояния для данного сценария.
- [dnsdist](https://github.com/PowerDNS/pdns) — мощный DNS load balancer с
  health checks. Для домашнего OpenWrt-роутера это более тяжёлая и широкая
  система, чем требуется.
- [AdGuard dnsproxy](https://github.com/AdguardTeam/dnsproxy) — качественная Go
  библиотека и proxy с UDP/TCP/DoH/DoT/DoQ/DNSCrypt. Его fallback применяется
  после ошибки основных upstream в текущем запросе; штатный режим не реализует
  требуемый persistent strict-priority state machine.

Вывод: разработка целесообразна, если сохранить узкую область проекта. Попытка
сразу конкурировать со SmartDNS, mosdns или AdGuard Home по протоколам,
фильтрации и маршрутизации неоправданно увеличит сложность.

## Почему Go

Go подходит для первой версии:

- хорошие сетевые и concurrency-примитивы;
- простой UDP/TCP DNS через
  [`miekg/dns`](https://github.com/miekg/dns);
- race detector, fuzzing и удобные интеграционные тесты;
- зрелая поддержка кросс-компиляции в OpenWrt;
- один daemon без интерпретатора и C runtime.

Цена выбора — бинарник и idle RSS обычно больше, чем у C/Rust. Поэтому в CI и
на реальных устройствах будут контролироваться размер, память и задержка.
Переход на Rust/C имеет смысл только после измерений, если Go не укладывается в
бюджет целевых роутеров.

## Почему не встраивать AdGuard dnsproxy целиком

Для MVP нужны обычные UDP/TCP upstream, включая локальный
`https-dns-proxy`. Полный AdGuard dnsproxy добавит протоколы и зависимости, но
не заменит основной алгоритм состояния:

- load-balance не равен строгому приоритету;
- fallback текущего запроса не означает запоминание отказа;
- health/failback hysteresis всё равно потребуется писать отдельно.

Поэтому MVP использует небольшой transport interface и `miekg/dns`. Когда
появится требование напрямую поддержать DoH/DoT/DoQ, можно отдельно оценить
переиспользование `github.com/AdguardTeam/dnsproxy/upstream`.

## Планируемое поведение

Состояния одного upstream:

```text
unknown -> healthy -> suspect -> down
              ^                    |
              `---- recovering <---'
```

Базовые правила:

1. Выбирается доступный upstream с наименьшим числовым `priority`.
2. На один upstream действует `attempt_timeout`.
3. На весь клиентский запрос действует `request_timeout`.
4. При timeout, сетевой ошибке, повреждённом ответе, `SERVFAIL` или `REFUSED`
   запрос пробуется на следующем upstream, если остался бюджет времени.
   Timeout/сетевая/протокольная ошибка влияет на общее health-состояние.
   `SERVFAIL` или `REFUSED` отдельного пользовательского запроса запускает
   fallback, но сам по себе не отключает сервер глобально.
5. `NXDOMAIN` является нормальным ответом и не означает отказ DNS.
6. Фоновые проверки помечают отказ до того, как его заметит пользовательский
   запрос.
7. Недоступные upstream проверяются реже, с ограниченным exponential backoff и
   jitter.
8. Возврат на основной DNS выполняется только после нескольких успешных
   проверок, чтобы избежать переключений туда-сюда.
9. Если все upstream помечены недоступными, proxy выполняет аварийную
   half-open попытку в порядке приоритета, а не возвращает мгновенный
   `SERVFAIL` по устаревшему состоянию.

Health check должен использовать настраиваемые домены. Успехом считается
корректный DNS-ответ на отправленный вопрос, а не обязательное совпадение с
заранее заданным IP. Для российских сетевых ограничений разумно настроить
несколько стабильных доменов, доступных в обычном режиме и при включённых белых
списках.

Health-состояние MVP хранится в RAM. После перезапуска daemon серверы начинают
в состоянии `unknown` и немедленно проверяются. Это исключает частые записи во
flash и перенос устаревшего состояния через reboot.

## Планируемая UCI-конфигурация

```uci
config main 'main'
        option enabled '1'
        option listen_addr '127.0.0.1'
        option listen_port '5359'
        option attempt_timeout_ms '700'
        option request_timeout_ms '2000'
        option health_interval_s '5'
        option fail_threshold '2'
        option recover_threshold '2'
        list probe 'example.com:A'
        list probe 'ya.ru:A'

config upstream 'encrypted_local'
        option enabled '1'
        option priority '10'
        option protocol 'udp'
        option address '127.0.0.1'
        option port '5054'

config upstream 'isp'
        option enabled '1'
        option priority '20'
        option protocol 'udp'
        option address '192.0.2.53'
        option port '53'

config upstream 'last_resort'
        option enabled '1'
        option priority '30'
        option protocol 'tcp'
        option address '198.51.100.53'
        option port '53'
```

До появления безопасного bootstrap-механизма рекомендуется задавать upstream
IP-адресами. Резолв hostname через dnsmasq, который сам направлен на этот proxy,
может создать DNS loop.

## Интеграция с dnsmasq

Рекомендуемая итоговая настройка dnsmasq:

```uci
config dnsmasq
        option noresolv '1'
        list server '127.0.0.1#5359'
```

Пакет не меняет dnsmasq автоматически при установке. Управление доступно в
LuCI и CLI:

```sh
failsafe-dns-proxy-dnsmasq dry-run
failsafe-dns-proxy-dnsmasq enable
failsafe-dns-proxy-dnsmasq status --json
failsafe-dns-proxy-dnsmasq disable
```

Механизм:

- сохранить текущие параметры dnsmasq;
- проверить конфигурацию и доступность proxy;
- применить `noresolv` и единственный upstream proxy;
- перезапустить сервисы в безопасном порядке;
- восстановить предыдущую настройку при ошибке или по команде пользователя.

Если после enable были внесены другие изменения в пакет UCI `dhcp`, disable
откажется автоматически накатывать старый backup, чтобы не потерять новые
настройки.

Без `noresolv` dnsmasq может параллельно использовать DNS из DHCP и обходить
алгоритм proxy.

## Компоненты

Планируются отдельные пакеты:

- `failsafe-dns-proxy` — Go daemon, CLI, UCI-конфиг и `procd` init script;
- `luci-app-failsafe-dns-proxy` — настройка, состояние upstream и управление
  интеграцией с dnsmasq;
- `luci-i18n-failsafe-dns-proxy-ru` — русская локализация.

CLI:

```text
failsafe-dns-proxy run
failsafe-dns-proxy check-config
failsafe-dns-proxy status
failsafe-dns-proxy status --json
failsafe-dns-proxy probe <upstream>
failsafe-dns-proxy self-test
```

Многодневный мониторинг:

```sh
failsafe-dns-proxy-soak start 168 60
failsafe-dns-proxy-soak status
failsafe-dns-proxy-soak report
failsafe-dns-proxy-soak stop
```

## LuCI

Интерфейс содержит:

- включение сервиса и адрес/порт listener;
- сортируемый список upstream с приоритетами;
- per-upstream protocol, address, port и enabled;
- таймауты, интервалы и пороги fail/recover;
- список probe-доменов;
- таблицу runtime-состояния: active, state, latency, last success, last error,
  consecutive failures/successes;
- кнопки «Проверить конфигурацию», «Проверить upstream», «Применить»;
- отдельное подтверждаемое управление dnsmasq.

LuCI только редактирует UCI и получает статус daemon. Алгоритм failover не
дублируется в JavaScript/Lua.

## Поддерживаемые OpenWrt

Минимальная версия — **OpenWrt 24.10.0**.

На 25 июня 2026 года актуальны две поддерживаемые стабильные серии:

- OpenWrt 24.10.x — IPK и `opkg`; серия находится в режиме old stable и
  ожидает завершения поддержки в сентябре 2026 года;
- OpenWrt 25.12.x — APK и `apk`.

OpenWrt 24.10 не устанавливает APK. OpenWrt 25.12 перешёл с `opkg` на `apk`,
поэтому release pipeline обязан собирать оба формата официальными SDK.

Daemon является user-space pure-Go приложением и не зависит от ABI ядра.
Сборки можно дедуплицировать по OpenWrt package architecture, а не повторять
для каждого устройства с той же архитектурой. Точная версия
target/subtarget всё равно хранится в release manifest и проверяется
установщиком.

## Автоматические сборки

Реализованы workflows:

- `ci.yml` — format, lint, unit, race и integration tests;
- `build-one.yml` — ручная сборка одной точной комбинации
  OpenWrt version/target/subtarget;
- `ci.yml` также выполняет package smoke build официальным SDK.

Reusable release matrix, APK и автоматическая публикация остаются P2.

В отличие от kernel module, нет необходимости собирать отдельный бинарник для
каждого устройства. Release matrix сначала получает все официальные
target/subtarget, определяет `pkgarch`, затем оставляет по одному SDK на
уникальную package architecture. LuCI собирается как `Architecture: all`.

За основу discovery и installer UX можно взять
[Slava-Shchipunov/awg-openwrt](https://github.com/Slava-Shchipunov/awg-openwrt),
но в этом проекте нельзя:

- подавлять ошибки сборки через `|| true`;
- искать SDK ненадёжным регулярным выражением без проверки checksum;
- устанавливать артефакт другого формата как скрытый fallback;
- публиковать release без manifest и SHA-256.

## Установщик

Локальная установка на поддерживаемом роутере:

```sh
./scripts/install.sh \
  --manifest ./build/supported-openwrt.json \
  --source-dir ./dist
```

Для явного включения dnsmasq добавляется `--configure-dnsmasq`. Без этого
флага DNS-конфигурация не меняется.

Установщик должен:

1. Получить version, target, subtarget и package architecture через
   `ubus call system board`.
2. Определить `opkg`/IPK или `apk`/APK.
3. Скачать точный release manifest.
4. Найти только совместимые daemon, LuCI и localization packages.
5. Проверить SHA-256 каждого файла.
6. Установить daemon, затем LuCI.
7. Не менять dnsmasq без отдельного флага или интерактивного подтверждения.
8. При любой ошибке оставить существующую DNS-конфигурацию рабочей.

## Оценка сложности

Оценка для одного разработчика при наличии тестовых роутеров:

| Этап | Сложность | Ориентир |
|---|---:|---:|
| DNS listener, UDP/TCP forwarding, deadlines | средняя | 3–5 дней |
| State machine, health checks, failover/failback | высокая | 5–8 дней |
| UCI, procd, reload/status | средняя | 3–5 дней |
| LuCI приложение и ACL | средняя | 4–7 дней |
| IPK/APK packaging и release matrix | высокая | 5–10 дней |
| Installer, manifests, checksums | средняя | 2–4 дня |
| Интеграционные тесты и реальные устройства | высокая | 5–10 дней |

Рабочий MVP реалистичен примерно за 4–7 недель. Стабильный публичный релиз под
все заявленные архитектуры потребует дополнительных тестов на слабых MIPS,
ARMv7 и ARM64 устройствах.

## Основные риски

- Неправильная классификация DNS-ответа может вызвать ложный failover.
- Слишком короткий timeout создаст ложные отказы на перегруженном роутере.
- Слишком агрессивный failback вызовет flapping.
- Одновременный restart dnsmasq и proxy может временно оставить клиентов без
  DNS.
- Upstream hostname без отдельного bootstrap DNS может создать loop.
- «Все платформы» нельзя честно заявить только по факту компиляции; нужны
  smoke tests хотя бы для основных семейств архитектур.
- Health check одного домена не отличает отказ DNS от блокировки конкретного
  домена.

Подробная последовательность реализации находится в
[`docs/PLAN.md`](docs/PLAN.md).

## Источники

- [OpenWrt 24.10.0 release notes](https://openwrt.org/releases/24.10/notes-24.10.0)
- [OpenWrt 25.12.0 release notes](https://openwrt.org/releases/25.12/notes-25.12.0)
- [OpenWrt downloads](https://downloads.openwrt.org/)
- [LuCI](https://github.com/openwrt/luci)
- [AdGuard dnsproxy](https://github.com/AdguardTeam/dnsproxy)
- [SmartDNS](https://github.com/pymumu/smartdns)
- [mosdns](https://github.com/IrineSistiana/mosdns)
- [PowerDNS/dnsdist](https://github.com/PowerDNS/pdns)
- [AWG OpenWrt build/install example](https://github.com/Slava-Shchipunov/awg-openwrt)
