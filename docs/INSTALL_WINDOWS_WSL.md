# Запуск на Windows через WSL 2

**Статус на 2026-10-18: маршрут описан, но проектная связка runner + cgroup v2 + seccomp на Windows/WSL здесь не запускалась. Это инструкция для проверки, не заявление о поддерживаемой платформе.** Публично исполнять пользовательский код нельзя.

Open Go Academy использует Linux rootless Podman, cgroup v2, systemd и seccomp. Запускайте весь репозиторий внутри Linux-дистрибутива WSL 2, не смешивайте Windows и Linux Node/Go процессы и не отключайте sandbox preflight. После bootstrap результат должен подтвердить локальная диагностика проекта; если rootless limits/seccomp не доступны, остановитесь и используйте проверенный Linux host.

## 1. Установите WSL 2

В PowerShell от администратора выполните официальную команду Microsoft:

```powershell
wsl --install
wsl --list --verbose
```

Windows 11 или Windows 10 build 19041+ указан Microsoft как prerequisite; новая установка по умолчанию использует WSL 2. Создайте Linux-пользователя при первом запуске дистрибутива. Документация Microsoft: [Install WSL](https://learn.microsoft.com/en-us/windows/wsl/install) и [Systemd in WSL](https://learn.microsoft.com/en-us/windows/wsl/systemd) (проверено 2026-10-18).

## 2. Подготовьте Linux-среду

Откройте Ubuntu (или другой совместимый дистрибутив) и следуйте разделу [Локальный запуск Linux](../README.md#локальный-запуск-linux). Положите clone в Linux home, например `~/src/open-go-academy`, а не в `/mnt/c`, чтобы Linux permissions и файловые операции runner соответствовали ожидаемой среде.

Если systemd не запущен, проверьте актуальную версию WSL и настройте `/etc/wsl.conf` по официальной инструкции Microsoft: добавьте или объедините секцию `[boot]` со строкой `systemd=true`, сохраните существующие другие секции, затем из PowerShell выполните `wsl --shutdown` и откройте дистрибутив снова. Не копируйте пример поверх существующего файла.

После установки проверьте:

```sh
uname -a
systemctl is-system-running
podman info --format '{{.Host.Security.Rootless}} {{.Host.CgroupsVersion}}'
podman info --debug
```

Первые команды должны подтвердить rootless Podman и cgroup v2; `podman info` отдельно не является проверкой project seccomp. Для seccomp, systemd integration и остальных ограничений выполняйте проектный sandbox preflight. Если он не подтверждает каждое требование, никакой host-execution fallback использовать нельзя. Системное состояние systemd само по себе не заменяет project preflight.

## 3. Установите и запустите проект

Из корня Linux clone выполните `make bootstrap`, `make build`, затем в двух WSL-терминалах `make runner` и `make dev`. Откройте `http://127.0.0.1:8380` в Windows-браузере. Сохранённые файлы, `.data/`, сокеты и ключи должны оставаться внутри Linux filesystem; не добавляйте `.data` в Git и не копируйте ключи на общий Windows каталог.

`make sandbox-test` содержит destructive rootless resource probes. Запускайте их только на disposable developer environment и сверяйте container labels перед cleanup.

## Ограничения

- Подтверждённая среда проекта — Fedora 44 Linux; Windows/WSL 2 не прошла установочную или sandbox-валидацию.
- Не обещаются performance, Podman systemd/cgroup поведение или готовность проекта на любом WSL дистрибутиве.
- Если `podman info`, seccomp, cgroup detection либо runner preflight не проходят, code execution должен оставаться выключенным. Не переключайте Podman на rootful и не снимайте ограничения ради запуска.
