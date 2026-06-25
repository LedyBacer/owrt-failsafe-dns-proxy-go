# Возможности

[English](../en/features.md) | **Русский**

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

[← Оглавление](index.md)
