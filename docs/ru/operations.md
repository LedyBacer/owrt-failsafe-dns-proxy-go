# Эксплуатация

[English](../en/operations.md) | **Русский**

## Основные команды

```sh
failsafe-dns-proxy version
failsafe-dns-proxy check-config
failsafe-dns-proxy status
failsafe-dns-proxy status --json
failsafe-dns-proxy probe encrypted_local
failsafe-dns-proxy probe --json public_fallback
failsafe-dns-proxy self-test

/etc/init.d/failsafe-dns-proxy status
/etc/init.d/failsafe-dns-proxy reload
logread -e failsafe-dns-proxy
```

Daemon не логирует каждый DNS-запрос. В журнал попадают operational events:
отказ, переход в recovery, восстановление, failover/failback, rejected reload
и emergency mode.

## Многодневный soak test

```sh
failsafe-dns-proxy-soak start 168 60
failsafe-dns-proxy-soak status
failsafe-dns-proxy-soak report
failsafe-dns-proxy-soak stop
```

Первый аргумент — длительность в часах, второй — интервал измерений в секундах.
Monitor записывает RSS, goroutines, file descriptors, heap, active upstream и
результат self-test в `/tmp/failsafe-dns-proxy-soak`.

## Отключение dnsmasq-интеграции

```sh
failsafe-dns-proxy-dnsmasq disable
```

Команда восстанавливает сохранённые параметры. Если UCI package `dhcp` был
изменён после включения интеграции, автоматическое восстановление будет
отклонено, чтобы не затереть более новые настройки.

[← Оглавление](index.md)
