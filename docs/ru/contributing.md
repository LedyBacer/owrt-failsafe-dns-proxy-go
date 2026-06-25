# Участие в проекте

[English](../en/contributing.md) | **Русский**

Проект развивается в свободное время и опирается на реальное использование на
роутерах. Любой вклад приветствуется: код, документация, packaging,
переводы, багрепорты и проверка на железе.

## Чем можно помочь

- **Улучшить daemon** — failover, health checks, производительность, тесты.
- **Доработать LuCI и docs** — понятнее UI, лучше гайды, новые переводы.
- **Починить packaging и CI** — OpenWrt recipes, installer, GitHub Actions.
- **Сообщать об ошибках** — шаги воспроизведения, логи, версия OpenWrt.
- **Поддерживать проект** — review pull request, triage issues, актуальные
  зависимости и workflows.

Перед существенным изменением прочитайте [PLAN.md](../PLAN.md) и
[AGENTS.md](../../AGENTS.md). Изменения поведения должны сопровождаться
тестами на минимальном полезном уровне и обновлением плана, если меняется
scope или статус.

Настройка разработки и проверки: [Разработка](development.md).

## Добавить протестированное устройство

Официальные release покрывают небольшую matrix. Многие комбинации собираются
через **Build one OpenWrt target** и проверяются пользователями на реальном
железе. Если daemon успешно отработал на вашем роутере, поделитесь результатом
— так другие поймут, что комбинацию можно пробовать.

### Что проверить

Минимум на вашем устройстве:

- установка пакета и запуск сервиса;
- отказ основного upstream и переход на резерв;
- recovery и failback на основной upstream;
- reload конфигурации и, при использовании, enable/disable интеграции с
  dnsmasq.

Желательно дополнительно: LuCI, upgrade/remove lifecycle, многодневный soak.

### Что указать в pull request или issue

| Поле | Пример |
| --- | --- |
| Устройство | Xiaomi Redmi Router AX6S |
| Версия OpenWrt | 24.10.7 |
| Target / subtarget | `mediatek` / `mt7622` |
| Package architecture | `aarch64_cortex-a53` |
| Формат пакета | IPK / `opkg` или APK / `apk` |
| Способ установки | release, fork build, локальная SDK-сборка |
| Что проверено | install, failover, failback, reload, … |

По возможности приложите вывод команд (`ubus call system board`,
`failsafe-dns-proxy status --json`, короткие фрагменты логов). Не публикуйте
приватные адреса, credentials и полные query logs.

### Какие файлы обновить

Для **зафиксированной проверки на железе** (без добавления в default release):

- [docs/en/compatibility.md](../en/compatibility.md) и
  [docs/ru/compatibility.md](compatibility.md) — строка или заметка в разделе
  community-tested devices;
- [docs/en/project-status.md](../en/project-status.md) и
  [docs/ru/project-status.md](project-status.md) — если результат заметно
  меняет картину готовности.

Для **нового default release target** (после review maintainer, со сборкой в
CI):

- [build/release-targets.json](../../build/release-targets.json) — честно
  выставить `tested_on_hardware`, точные version/target/subtarget;
- compatibility и project-status;
- [PLAN.md](../PLAN.md) — статус release и verification.

Не помечайте target как tested on hardware, если сценарии выше не прогнаны на
этом устройстве. Только compile/smoke-build — в compatibility notes, без
claims об аппаратной проверке.

## Pull requests

1. Сделайте fork и отдельную ветку с узкой задачей.
2. Локальные проверки: `make lint` и `make ci` при изменениях Go или shell.
3. Опишите что, зачем и как тестировали.
4. Не смешивайте несвязанные правки в одном pull request.

Вопросы и WIP — через draft pull request или issue.

## Правила общения

Будьте конструктивны и точны. Security issues — приватно, см.
[Безопасность](security.md).

[← Оглавление](index.md)
