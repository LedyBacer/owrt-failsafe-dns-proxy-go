# Быстрый запуск

[English](../en/quick-start.md) | **Русский**

1. Откройте LuCI → **Services → Failsafe DNS Proxy** или измените
   `/etc/config/failsafe-dns-proxy`.
1. Замените примерные upstream своими адресами.
1. Включите daemon и проверьте конфигурацию:

```sh
uci set failsafe-dns-proxy.main.enabled='1'
uci commit failsafe-dns-proxy

failsafe-dns-proxy check-config
/etc/init.d/failsafe-dns-proxy enable
/etc/init.d/failsafe-dns-proxy start
failsafe-dns-proxy self-test
failsafe-dns-proxy status
```

1. Только после успешного self-test направьте dnsmasq на proxy:

```sh
failsafe-dns-proxy-dnsmasq dry-run
failsafe-dns-proxy-dnsmasq enable
failsafe-dns-proxy-dnsmasq status
```

Команда `enable` сохраняет предыдущую UCI-конфигурацию dnsmasq, применяет
`noresolv=1` и `server=127.0.0.1#5359`, проверяет DNS и выполняет rollback при
ошибке.

[← Оглавление](index.md)
