# План реализации

## Правила ведения плана

- Этот файл является основным roadmap и журналом статуса проекта.
- Перед существенным изменением нужно сверить его с текущей реализацией.
- Выполненные, добавленные, перенесённые и заблокированные задачи обновляются
  здесь в том же patch.
- `[x]` означает, что задача реализована и проверена на достаточном уровне;
  `[~]` — реализована частично; `[ ]` — ещё не выполнена.

## Текущий статус и приоритеты

### P0 — критично перед постоянным использованием

- [x] безопасная явная интеграция с dnsmasq: backup, dry-run, enable, disable,
  health verification и rollback;
- [x] atomic runtime reload без потери listeners и с сохранением старой
  конфигурации при ошибке;
- [x] событийные operational logs: upstream down/recovering/recovered,
  failover/failback и emergency half-open без логирования каждого запроса;
- [~] длительный soak test на роутере: короткий stress-soak пройден без
  goroutine/FD leak, автономный 7-дневный монитор запущен 25 июня 2026 года;
- [x] package upgrade/remove tests с сохранением UCI и безопасным состоянием
  dnsmasq.

### P1 — важно для управляемого домашнего использования

- [x] LuCI application: конфигурация, статус, service controls, тестирование
  upstream и просмотр operational logs;
- [x] `luci-i18n-failsafe-dns-proxy-ru`;
- [x] least-privilege RPC/ACL без произвольного shell access;
- [x] installer с exact compatibility и checksum verification;
- [x] CI для unit/race/fuzz/package smoke checks.

### P2 — после стабилизации MVP

- [ ] release manifest и GitHub Actions release pipeline;
- [ ] APK/OpenWrt 25.12+;
- [ ] расширенная matrix targets;
- [ ] native encrypted transports и loop-safe hostname bootstrap;
- [ ] дополнительные LuCI локализации.

## Статус первого MVP

На 25 июня 2026 года реализован первый вертикальный срез:

- [x] daemon и CLI;
- [x] UDP/TCP listener и upstream transport;
- [x] truncated UDP retry по TCP;
- [x] strict-priority state machine, passive failures, active probes, hysteresis,
  emergency half-open и bounded backoff с jitter;
- [x] UCI parser/validator;
- [x] read-only status socket;
- [x] procd/UCI package files;
- [x] официальный IPK для OpenWrt 24.10.7 `mediatek/mt7622`
  (`aarch64_cortex-a53`);
- [x] unit/integration/race/fuzz smoke tests;
- [x] установка и failover/recovery/failback тест на Xiaomi Redmi Router AX6S;
- [x] LuCI, русская локализация и narrow rpcd API установлены и проверены на
  Xiaomi Redmi Router AX6S;
- [x] operational log viewer проверен реальным отказом primary upstream:
  записаны `upstream unavailable` и переключение на резервный;
- [x] atomic SIGHUP reload проверен под DNS-нагрузкой: PID/listeners не
  изменились, ошибочная конфигурация и смена listener были отклонены;
- [x] dnsmasq enable/disable и rollback проверены на реальной конфигурации
  `https-dns-proxy`, исходный UCI export восстановлен без изменений;
- [x] disable/remove отказывается затирать DHCP-изменения, сделанные после
  включения интеграции;
- [x] remove/reinstall lifecycle проверен: prerm восстановил dnsmasq, UCI
  сохранился, installer выбрал точный artifact и проверил SHA-256;
- [x] короткий stress-soak: 120 запросов, 0 ошибок, goroutine 10→9, FD 11→10;
- [x] измерение на роутере: RSS около 5.5 MiB, бинарник около 4.9 MiB.

Текущий этап: автономный 7-дневный домашний soak, затем анализ отчёта и P2.

## 0. Зафиксированные решения

- Язык daemon: Go.
- DNS-библиотека MVP: `github.com/miekg/dns`.
- Клиентские протоколы: UDP и TCP.
- Upstream-протоколы MVP: UDP и TCP.
- dnsmasq остаётся LAN resolver/cache и направляет запросы в proxy.
- Алгоритм: strict priority с persistent health state.
- Конфигурация: UCI.
- Управление процессом: procd.
- UI: отдельный `luci-app-failsafe-dns-proxy`.
- OpenWrt: 24.10.0+, IPK для 24.10 и APK для 25.12+.
- Сборка пакетов: только официальными OpenWrt SDK.
- Полный AdGuard dnsproxy не используется в MVP.

## 1. Discovery prototype

Цель: проверить алгоритм до упаковки в OpenWrt.

Работы:

- [x] создать Go module и минимальный CLI;
- [x] определить `Upstream` interface;
- [x] реализовать UDP/TCP exchange с deadline;
- [x] поднять UDP/TCP listener;
- [x] добавить fake upstream для success, timeout, malformed, SERVFAIL и delayed
  response;
- [x] измерить stripped binary и idle RSS на реальном роутере.

Критерии готовности:

- запрос проходит через proxy;
- UDP truncation повторяется по TCP;
- timeout ограничен и не зависает;
- `go test -race ./...` проходит;
- есть исходные измерения размера и памяти.

## 2. Failover state machine

Цель: реализовать главное отличие продукта.

Работы:

- [x] состояния `unknown`, `healthy`, `suspect`, `down`, `recovering`;
- [x] strict-priority selector;
- [x] failure/recovery counters;
- [x] passive updates по реальным запросам;
- [x] active probe scheduler;
- [x] разделение query fallback и global health evidence: единичный
  `SERVFAIL`/`REFUSED` не отключает upstream глобально;
- [x] backoff и jitter для down upstream;
- [x] emergency half-open path, когда все upstream помечены недоступными;
- [x] единый total request budget для нескольких попыток;
- [x] защита от stampede: не более 16 одновременных попыток на один upstream;
- [x] immutable runtime snapshots и atomic live reload.

Критерии готовности:

- после подтверждённого отказа новые запросы не ждут timeout этого upstream;
- текущий запрос пробует резервный сервер в пределах total deadline;
- восстановление требует заданного числа успехов;
- более приоритетный восстановившийся upstream автоматически возвращается;
- тесты используют fake clock и не зависят от `time.Sleep`.

## 3. DNS correctness and resilience

Работы:

- [x] проверка ID/question/response shape;
- [x] явная классификация RCODE;
- [x] сохранение EDNS данных; расширенные тесты больших UDP-пакетов ещё нужны;
- [x] TCP framing через `miekg/dns`;
- [x] лимит concurrent requests;
- [x] graceful shutdown;
- [x] atomic reload без потери listener;
- [x] fuzz smoke test response parser/validator;
- [x] structured operational logs без query logging: failure/down/recovery,
  failover/failback и emergency mode.

Критерии готовности:

- malformed/mismatched ответы не попадают клиенту;
- `NXDOMAIN` не снижает health;
- passive `SERVFAIL`/`REFUSED` запускает fallback, но не отравляет global
  health одним запросом;
- probe `SERVFAIL`/`REFUSED` считается неуспешной проверкой;
- shutdown и reload не оставляют goroutine/listener leaks.

## 4. UCI, procd and status API

Работы:

- [x] схема `/etc/config/failsafe-dns-proxy`;
- [x] parser/validator UCI;
- [x] `check-config`;
- [x] `status --json`;
- [x] read-only Unix status socket;
- [x] procd init script с respawn и reload trigger;
- [x] logread-friendly operational transition messages без query logging;
- [x] package defaults и conffiles.

Критерии готовности:

- неверный reload не заменяет рабочую конфигурацию;
- procd корректно перезапускает daemon;
- runtime status содержит active upstream и состояние каждого upstream;
- конфигурация сохраняется при обновлении пакета.

## 5. dnsmasq integration

Работы:

- [x] read/backup полной UCI-конфигурации `dhcp`;
- [x] dry-run проверки;
- [x] явные enable/disable команды интеграции;
- [x] безопасный restart и ожидание готовности dnsmasq;
- [x] rollback при неуспешной проверке;
- [x] защита от loopback upstream, совпадающего с listener.

Критерии готовности:

- install пакета не меняет dnsmasq;
- enable переводит dnsmasq на единственный proxy upstream;
- disable восстанавливает сохранённые параметры;
- при ошибке proxy DNS-настройки автоматически откатываются.

## 6. LuCI application

Работы:

- [x] JS LuCI view;
- [x] dynamic list upstream;
- [x] сортировка/редактирование priority;
- [x] validation адресов, портов, timeout и thresholds;
- [x] runtime status table;
- [x] probe/test actions;
- [x] service controls;
- [x] operational log viewer;
- [x] отдельный блок управления интеграцией dnsmasq;
- [x] narrow rpcd exec API и least-privilege ACL;
- [x] английская база и русская локализация;
- [~] HTTP/RPC/resource smoke пройден; визуальная приёмка desktop/mobile
  остаётся частью домашнего тестирования.

Критерии готовности:

- все настройки доступны без ручного редактирования UCI;
- UI не получает произвольный shell access;
- ошибки конфигурации показаны до restart;
- статус обновляется без перезагрузки страницы.

## 7. OpenWrt packages

Структура:

```text
package/failsafe-dns-proxy/
  Makefile
  files/etc/config/failsafe-dns-proxy
  files/etc/init.d/failsafe-dns-proxy

package/luci-app-failsafe-dns-proxy/
  Makefile
  root/usr/share/rpcd/acl.d/
  root/usr/share/luci/menu.d/
  htdocs/luci-static/resources/view/
  po/
```

Работы:

- [x] OpenWrt Go package Makefile;
- [x] reproducible flags: `-trimpath`, stripped symbols, version metadata;
- [x] IPK smoke build daemon/LuCI/i18n на 24.10;
- [ ] APK smoke build на 25.12;
- [x] package content assertions;
- [x] install/remove/upgrade tests на AX6S с сохранением UCI и dnsmasq.

Критерии готовности:

- пакеты собираются официальными SDK;
- daemon запускается после установки;
- LuCI menu и ACL работают;
- upgrade не стирает UCI;
- package format соответствует версии OpenWrt.

## 8. GitHub Actions

### `ci.yml`

- [x] `gofmt`/`go vet`;
- [x] unit/integration;
- [x] race;
- [x] fuzz smoke;
- [x] shellcheck;
- [x] LuCI/static checks;
- [x] официальный OpenWrt SDK package smoke build.

### `build-one.yml`

Inputs:

- `openwrt_version`;
- `target`;
- `subtarget`;
- optional `package_set`;
- optional `publish_artifact`.

Workflow проверяет существование официального SDK и собирает только точную
комбинацию.

### `build-openwrt.yml`

Reusable workflow:

- получает SDK URL из официального directory/index metadata;
- проверяет checksum;
- кэширует SDK по version/target/subtarget/checksum;
- устанавливает feeds;
- собирает package без подавления ошибок;
- проверяет содержимое;
- возвращает artifacts и metadata.

### `release.yml`

- запускается по version tag проекта;
- читает `build/supported-openwrt.json`;
- получает все target/subtarget для каждой поддерживаемой версии;
- вычисляет package architecture;
- дедуплицирует daemon jobs по `version + pkgarch`;
- отдельно собирает `Architecture: all` LuCI;
- формирует `manifest.json`, `SHA256SUMS` и release notes;
- публикует IPK/APK.

### `refresh-openwrt.yml`

- периодически проверяет новые stable patch releases;
- валидирует SDK indexes;
- обновляет supported manifest через pull request;
- не публикует поддержку автоматически без успешной сборки.

Критерии готовности:

- manual build работает для одного target/subtarget;
- release содержит все уникальные package architectures;
- каждый artifact перечислен в manifest и checksum;
- failed matrix job делает release неуспешным.

## 9. Installer

Файлы:

- `scripts/install.sh`;
- `scripts/lib/openwrt-detect.sh`;
- JSON schema/format release manifest.

Работы:

- [x] определить board/version/target/subtarget/pkgarch;
- [x] определить package manager;
- [x] выбрать exact-compatible artifacts;
- [x] скачать во временный каталог или использовать локальный каталог;
- [x] проверить SHA-256 до установки;
- [x] установить daemon, LuCI, localization;
- [x] optional `--configure-dnsmasq`;
- [x] non-interactive flags и daemon-only режим;
- [x] понятные коды выхода;
- [x] очистка временных файлов trap-ом.

Критерии готовности:

- 24.10 выбирает только IPK;
- 25.12+ выбирает только APK;
- несовместимая платформа завершается до изменений;
- checksum mismatch блокирует установку;
- dnsmasq не меняется без явного согласия.

## 10. Validation matrix

Минимальные реальные/эмулированные семейства:

- `x86_64`;
- `aarch64_cortex-a53` или близкое ARM64;
- `arm_cortex-a7_neon-vfpv4` или близкое ARMv7;
- `mips_24kc`;
- `mipsel_24kc`.

Сценарии:

- основной upstream выключен до старта;
- основной пропадает во время нагрузки;
- основной отвечает медленнее timeout;
- ответы теряются выборочно;
- резервный тоже недоступен;
- основной восстанавливается и flaps;
- reload во время запросов;
- restart dnsmasq/proxy в разных порядках;
- local `https-dns-proxy` на `127.0.0.1`;
- DNS провайдера по адресу из пользовательской конфигурации.

## 11. Отложенные функции

Не включать в MVP без отдельного решения:

- native DoH/DoT/DoQ;
- автоматическое получение DNS провайдера из DHCP/interface;
- per-domain routing;
- cache;
- DNSSEC validation;
- blocklists;
- Prometheus endpoint;
- active-active load balancing;
- использование ответа «самого быстрого» DNS;
- автоматическая подмена системного bootstrap resolver.

Наиболее полезное расширение после MVP — динамический upstream типа
`network:<interface>`, который безопасно читает DNS, полученный WAN по DHCP, и
обновляет список без DNS loop.

## 12. Решения, которые нужно подтвердить перед кодом

1. Default listener port: оставить `5359` как на исходном приложении или выбрать
   другой.
2. Default passive policy: помечать `down` после первого timeout или после двух
   последовательных ошибок.
3. Нужен ли dynamic ISP DNS из WAN DHCP уже в MVP.
4. Должен ли installer по умолчанию ставить только daemon или daemon + LuCI.
5. Нужны ли OpenWrt forks в первой публичной версии или только официальный
   OpenWrt.
