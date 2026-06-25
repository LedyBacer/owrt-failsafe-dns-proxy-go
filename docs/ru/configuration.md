# Конфигурация

[English](../en/configuration.md) | **Русский**

## Настройки, поставляемые пакетом

```uci
config main 'main'
        option enabled '0'
        option listen_addr '127.0.0.1'
        option listen_port '5359'
        option attempt_timeout_ms '700'
        option request_timeout_ms '2000'
        option health_interval_s '5'
        option fail_threshold '2'
        option recover_threshold '2'
        option max_concurrent '128'
        option status_socket '/var/run/failsafe-dns-proxy.sock'
        list probe 'example.com:A'
        list probe 'ya.ru:A'

config upstream 'encrypted_local'
        option enabled '1'
        option priority '10'
        option protocol 'udp'
        option address '127.0.0.1'
        option port '5054'

config upstream 'public_fallback'
        option enabled '1'
        option priority '20'
        option protocol 'udp'
        option address '77.88.8.8'
        option port '53'
```

`enabled` намеренно равен `0`: пакет не должен запускать непроверенную
конфигурацию автоматически.

На тестовом AX6S сохранена эта же схема, но daemon включён:
`failsafe-dns-proxy.main.enabled=1`. Основной upstream — локальный
`https-dns-proxy` на `127.0.0.1:5054`, резервный — `77.88.8.8:53`.

## Значение параметров

| Параметр | Сейчас | Назначение |
| --- | ---: | --- |
| `listen_addr` | `127.0.0.1` | адрес listener; loopback безопасен для схемы с dnsmasq |
| `listen_port` | `5359` | UDP и TCP порт proxy |
| `attempt_timeout_ms` | `700` | максимум одной попытки к upstream |
| `request_timeout_ms` | `2000` | общий бюджет запроса со всеми fallback |
| `health_interval_s` | `5` | базовый интервал active probes |
| `fail_threshold` | `2` | число подтверждённых ошибок до состояния `down` |
| `recover_threshold` | `2` | число успешных probes до восстановления |
| `max_concurrent` | `128` | общий предел одновременно обрабатываемых запросов |
| `probe` | два DNS-вопроса | вопросы для фоновой проверки транспорта |
| `priority` | `10`, `20` | меньшее число означает более высокий приоритет |
| `protocol` | `udp` | plain DNS transport; допустимы `udp` и `tcp` |

`request_timeout_ms` должен быть не меньше `attempt_timeout_ms`. Если
upstream-серверов несколько, общий бюджет должен оставлять время хотя бы для
одной резервной попытки.

## Как изменить настройки

### Через LuCI

Откройте **Services → Failsafe DNS Proxy**. Интерфейс позволяет:

- включить сервис и изменить listener;
- добавить, отключить или удалить upstream;
- изменить приоритет, протокол, IP и порт;
- настроить таймауты, thresholds и probe-вопросы;
- проверить конфигурацию и отдельный upstream;
- увидеть активный upstream, health state и последние ошибки;
- включить или отключить интеграцию с dnsmasq.

### Через UCI

Пример замены резервного DNS:

```sh
uci set failsafe-dns-proxy.public_fallback.address='1.1.1.1'
uci set failsafe-dns-proxy.public_fallback.port='53'
uci commit failsafe-dns-proxy

failsafe-dns-proxy check-config
/etc/init.d/failsafe-dns-proxy reload
```

Добавление ещё одного upstream:

```sh
uci set failsafe-dns-proxy.last_resort='upstream'
uci set failsafe-dns-proxy.last_resort.enabled='1'
uci set failsafe-dns-proxy.last_resort.priority='30'
uci set failsafe-dns-proxy.last_resort.protocol='tcp'
uci set failsafe-dns-proxy.last_resort.address='8.8.8.8'
uci set failsafe-dns-proxy.last_resort.port='53'
uci commit failsafe-dns-proxy

failsafe-dns-proxy check-config
/etc/init.d/failsafe-dns-proxy reload
```

Имена upstream и значения `priority` должны быть уникальны. Адрес upstream
пока обязан быть IP-адресом. Hostname может создать bootstrap loop, если
dnsmasq уже направлен на этот proxy.

Reload применяется атомарно. Ошибочная конфигурация отклоняется, а daemon
продолжает работать со старой. Изменение `listen_addr`, `listen_port` или
`status_socket` требует restart:

```sh
/etc/init.d/failsafe-dns-proxy restart
```

[← Оглавление](index.md)
