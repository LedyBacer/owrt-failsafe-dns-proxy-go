# Совместимость

[English](../en/compatibility.md) | **Русский**

| OpenWrt | Формат | Менеджер | Статус |
| --- | --- | --- | --- |
| 24.10.7 | IPK | `opkg` | default release build для `mediatek/mt7622`, протестировано на AX6S |
| 25.12.4 | APK | `apk` | default release build для `mediatek/mt7622`, package smoke test |

Release matrix описана в
[build/release-targets.json](../../build/release-targets.json). Installer
принимает только точное совпадение version, target, subtarget, package
architecture и package manager.

Основной Release намеренно ограничен роутером Xiaomi Redmi Router AX6S
(`mediatek/mt7622`) и двумя точными версиями OpenWrt: `24.10.7` и `25.12.4`.
Для любой другой версии или платформы пользователь собирает собственный
artifact в GitHub fork через workflow **Build one OpenWrt target**. См.
[Сборка под свою платформу](../README.ru.md#сборка-под-свою-платформу).

## Устройства, проверенные сообществом

Проверки на железе вне default release matrix фиксируются здесь. Чтобы
добавить своё устройство, см.
[Участие в проекте → Добавить протестированное устройство](contributing.md#добавить-протестированное-устройство).

| Устройство | OpenWrt | Target | Проверено проектом | Заметки |
| --- | --- | --- | --- | --- |
| Xiaomi Redmi Router AX6S | 24.10.7 | `mediatek/mt7622` | да | эталонное устройство; полный MVP-сценарий |

Если вашей комбинации нет в таблице, успешная fork-сборка и тесты из
руководства для contributors достаточны, чтобы предложить запись.

[← Оглавление](index.md)
