# Запуск на macOS через Linux VM

**Статус на 2026-10-18: native macOS не поддерживается; предложен непроверенный путь через отдельный Linux VM. Проект не запускался на macOS в этой проверке.** Публичное исполнение пользовательского кода запрещено.

Open Go Academy ожидает Linux API/runner host, rootless Podman, cgroup v2, systemd и seccomp. Контейнеры используют Linux kernel. Официальный Podman требует Linux VM для своих контейнеров на macOS; это не подтверждает, что project runner, сокеты, cgroup limits и cleanup Open Go Academy корректны через удалённый Podman service.

## Рекомендуемый изолированный маршрут

1. Создайте отдельную Linux VM (например, Fedora 44 — единственная проверенная здесь базовая ОС) в выбранном вами гипервизоре. Ограничьте ресурсы VM и включите актуальные security updates.
2. Клонируйте репозиторий внутрь Linux filesystem VM, установите Linux prerequisites согласно [README](../README.md#локальный-запуск-linux) и [Началу работы](../GETTING_STARTED.md).
3. В Linux VM подтвердите rootless Podman, cgroup v2, systemd и seccomp; затем запустите проектный preflight. Если любое ограничение недоступно, оставьте runner выключенным.
4. Выполняйте `make bootstrap`, `make build`, `make runner` и `make dev` внутри Linux VM. Откройте `http://127.0.0.1:8380` в браузере macOS только через loopback-forwarding выбранной VM; не публикуйте API на LAN.
5. Храните `.data/`, приватные ключи и backups только в защищённом диске VM. Не размещайте source/data на общей host-mounted папке с ослабленными Linux permissions.

Официальный Podman описывает `podman machine` и виртуализированный Linux engine для macOS/Windows ([Podman installation](https://podman.io/docs/installation), [podman machine](https://docs.podman.io/en/latest/markdown/podman-machine.1.html); проверено 2026-10-18). Данная инструкция намеренно рекомендует отдельную Linux VM: runner использует локальные Linux процессы, сокеты и контроллеры cgroup, а интеграция с Podman machine не тестировалась.

## Ограничения

- Подтверждённая среда проекта — Fedora 44 Linux; macOS/VM маршрут не прошёл bootstrap или runner tests.
- Не обещаются native API, runner или database support на macOS.
- Не используйте Podman rootful, небезопасные флаги или host execution для обхода preflight.
