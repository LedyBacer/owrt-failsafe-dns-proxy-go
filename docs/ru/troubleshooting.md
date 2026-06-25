# Диагностика

[English](../en/troubleshooting.md) | **Русский**

## Сервис не запускается

```sh
failsafe-dns-proxy check-config
logread -e failsafe-dns-proxy
```

Проверьте `enabled=1`, уникальные priority, IP-адреса upstream и соотношение
таймаутов.

## Status socket недоступен

Проверьте, что daemon запущен и `status_socket` совпадает с CLI:

```sh
failsafe-dns-proxy status \
  --socket /var/run/failsafe-dns-proxy.sock
```

## После настройки клиенты обходят proxy

Проверьте dnsmasq:

```sh
uci show dhcp.@dnsmasq[0].noresolv
uci show dhcp.@dnsmasq[0].server
failsafe-dns-proxy-dnsmasq status
```

Для строгой схемы нужен `noresolv=1`; иначе dnsmasq может использовать DNS из
DHCP параллельно.

## Fallback слишком медленный

Уменьшайте `attempt_timeout_ms` осторожно и только после измерений. Слишком
малое значение создаёт ложные отказы на загруженном роутере или медленном
канале.

[← Оглавление](index.md)
