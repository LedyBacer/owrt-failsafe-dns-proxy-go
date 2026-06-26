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

## После перевода dnsmasq сервис периодически зависает

Если в логах есть timeout к локальному upstream вроде `127.0.0.1:5053` или
`127.0.0.1:5054`, а этот порт обслуживает `https-dns-proxy`, проверьте, что
`https-dns-proxy` сам больше не переписывает dnsmasq:

```sh
uci -q get https-dns-proxy.config.dnsmasq_config_update
uci set https-dns-proxy.config.dnsmasq_config_update='-'
uci commit https-dns-proxy
/etc/init.d/https-dns-proxy restart
failsafe-dns-proxy-dnsmasq dry-run
failsafe-dns-proxy-dnsmasq enable
```

У `https-dns-proxy` по умолчанию есть собственная интеграция с dnsmasq. Если
она включена одновременно с интеграцией Failsafe DNS Proxy, оба сервиса могут
менять `/etc/config/dhcp`, а локальный DoH upstream начинает отвечать
таймаутами после restart, WAN event или heartbeat.

## Fallback слишком медленный

Уменьшайте `attempt_timeout_ms` осторожно и только после измерений. Слишком
малое значение создаёт ложные отказы на загруженном роутере или медленном
канале.

[← Оглавление](index.md)
