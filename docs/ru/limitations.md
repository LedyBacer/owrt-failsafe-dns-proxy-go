# Ограничения

[English](../en/limitations.md) | **Русский**

- Native DoH, DoT и DoQ пока не реализованы.
- Upstream должен быть IP-адресом; loop-safe hostname bootstrap отсутствует.
- Нет DNS cache, filtering, DNSSEC validation, split DNS и load balancing.
- dnsmasq остаётся обязательным для LAN cache и локальных имён.
- APK-сборка не означает аппаратную проверку на каждом target.
- Поддерживается официальный OpenWrt; forks требуют отдельной валидации.

[← Оглавление](index.md)
