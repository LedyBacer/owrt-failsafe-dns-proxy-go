# Поведение failover

[English](../en/failover.md) | **Русский**

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

[← Оглавление](index.md)
