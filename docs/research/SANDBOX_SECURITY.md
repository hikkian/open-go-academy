# Изоляция исполнения Go

Проверено 2026-10-15. Область: MASTER_SPEC D2; это исследование и проектные gates, не заключение о безопасности ещё не созданного runner.

## Выбор и первичные источники

| Подход | Возможности и ограничения | Решение |
|---|---|---|
| bubblewrap + namespaces | Создаёт окружение из явно разрешённых mounts; готовой security policy нет. `--unshare-all`, `--new-session`, `--die-with-parent`, read-only toolchain, закрытые capabilities и фильтр syscalls требуют собственной политики. Общий kernel остаётся границей доверия. [README](https://github.com/containers/bubblewrap) | Условный local single-user backend; не публичная multi-tenant защита |
| seccomp | Уменьшает поверхность системных вызовов; сам по себе не sandbox. Нужны проверка архитектуры syscall и совместимость Go compiler/runtime. [Kernel seccomp](https://docs.kernel.org/userspace-api/seccomp_filter.html) | Обязательный слой публичной политики, gate локального полного профиля |
| cgroup v2 | Per-job `memory.max`, `memory.swap.max`, `pids.max`, `cpu.max`; доступные root controllers не доказывают делегацию worker. Для завершения всей группы — `cgroup.kill`, проверка `populated=0`. [Kernel cgroup v2](https://docs.kernel.org/admin-guide/cgroup-v2.html) | Проверять реальное применение до запуска пользовательского кода |
| rootless Podman | OCI rootfs и штатные resource flags; нужна работоспособная делегация cgroups. Network по умолчанию может быть доступна — задавать `--network=none`. [podman-run](https://docs.podman.io/en/latest/markdown/podman-run.1.html) | Выбранный local backend: реальные лимиты подтверждены probe ниже |
| rootless Docker | Daemon и контейнеры непривилегированны; `userns-remap` не равен rootless daemon. [Rootless](https://docs.docker.com/engine/security/rootless/) Resource flags требуют cgroup v2/systemd. [Ограничения](https://docs.docker.com/engine/security/rootless/troubleshoot/) | Только проверенный rootless context, без rootful fallback |
| gVisor | Sentry перехватывает API ОС и сокращает прямую kernel surface. Resource limits и сеть остаются обязанностью host; side channels не устранены. [Security model](https://gvisor.dev/docs/architecture_guide/security/) | Кандидат public backend после compatibility/negative tests |
| Firecracker | MicroVM с отдельным guest kernel; нужны KVM, jailer, непривилегированные UID/GID, seccomp, host hardening и жизненный цикл microVM. [Production host](https://github.com/firecracker-microvm/firecracker/blob/main/docs/prod-host-setup.md) | Альтернатива public backend; заметно дороже эксплуатационно |

Это инженерный выбор проекта: до подтверждения полного профиля runner возвращает `unavailable`, а не запускает код обычным process. Ни Docker по умолчанию, ни установленный bwrap не дают автоматического допуска.

Решение M0: только rootless Podman, отдельный `cmd/runner` с private Unix socket. API не принимает container arguments/image/mounts от клиента; public runner disabled. Bubblewrap исследован как альтернатива, но не используется как fallback. Документ не утверждает, что реализация worker уже существует.

## Фактическое окружение

Команды выполнены в этой сессии, без изменения host security settings:

- `uname -sr`: `Linux 7.2.10-200.fc44.x86_64`.
- `bwrap --version`: `bubblewrap 0.12.0`; `podman --version`: `5.8.7`; `docker --version`: `29.9.0`.
- `stat -fc %T /sys/fs/cgroup`: `cgroup2fs`; root controllers включают `cpu memory pids`.
- `podman info --format '{{.Host.CgroupsVersion}} {{.Host.CgroupManager}} {{.Host.Security.Rootless}}'`: значения `v2 systemd true` наблюдались в составном запросе; дополнительные запросы полей `SeccompEnabled` и `Seccomp` завершились template error, поэтому seccomp Podman не подтверждён.
- `docker info --format '{{json .SecurityOptions}}'`: `["name=seccomp,profile=builtin","name=cgroupns"]`; `rootless` отсутствует. Текущий daemon не считать rootless.
- Минимальный namespace probe ниже завершился exit 0. Он не применял seccomp/resource limits, не компилировал пользовательский Go и не является готовым профилем.

Главный агент сообщил фактически выполненный rootless Podman probe (2026-10-15):

```sh
podman run --rm --pull=never --network=none --read-only --cap-drop=ALL \
  --security-opt=no-new-privileges --user=65532:65532 \
  --memory=512m --memory-swap=512m --pids-limit=64 --cpus=1 \
  --tmpfs=/tmp:rw,noexec,nosuid,nodev,size=128m \
  --tmpfs=/work:rw,nosuid,nodev,size=128m --workdir=/work \
  docker.io/library/golang:1.27.2-alpine sh -c \
  'cat /sys/fs/cgroup/memory.max /sys/fs/cgroup/pids.max /sys/fs/cgroup/cpu.max; go version; id'
```

Результат: `536870912`, `64`, `100000 100000`, `go1.27.2`, `uid=65532`. Это подтверждает применение memory/PID/CPU flags и запуск Go toolchain; негативные tests/seccomp effectiveness/cleanup ещё не подтверждены. Для реализации закрепить локально подтверждённый digest: `docker.io/library/golang@sha256:3b15ba438c60ab8fabe5fee3b11025c0049e358b31feff384648f11fc8e335f8`. Использовать суммарный scratch budget: два tmpfs по 128 MiB дают до 256 MiB, а не единый лимит 128 MiB.

```sh
bwrap --unshare-all --die-with-parent --new-session \
  --ro-bind /usr /usr --symlink usr/lib64 /lib64 --symlink usr/lib /lib \
  --proc /proc --dev /dev --clearenv --setenv PATH /usr/bin /usr/bin/true
```

## Обязательный контракт local runner

Предлагаемые начальные бюджеты (нужна калибровка compiler): concurrency 1, queue 4, source 64 KiB/16 файлов, stdout+stderr 64 KiB, per-job memory 512 MiB, swap 0, tasks 128, CPU quota 1 core, wall time 15 s, scratch tmpfs 128 MiB. Бюджеты включают build/test и дочерние процессы; изменение только доверенной конфигурацией.

Выделенный worker с минимальным окружением получает validated files, а не shell command/mounts/runtime flags от клиента. Compiler и binary исполняются внутри одной границы; `CGO_ENABLED=0`, `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, отдельные ограниченные GOCACHE/GOTMPDIR. Toolchain фиксирован; rootfs read-only и минимален. `/home`, репозиторий, DB, issuer keys, `/run/user`, container sockets и host `/proc` недоступны; workspace per-job, без reuse и symlinks от клиента.

Лимиты одного process (`RLIMIT_AS`, `RLIMIT_FSIZE`) не заменяют aggregate бюджет: дочерние процессы/много файлов обходят такие ожидания; RLIMIT_AS требует проверки с виртуальными mappings Go. `RLIMIT_NPROC` относится к real UID и может затронуть другие процессы владельца. Process timeout без убийства всей группы и cleanup недостаточен. Бюджет tmpfs включает cache и все writable paths.

Fail closed при недоступных namespaces/cgroups/seccomp, ошибке запуска/проверки политики. Preflight должен проверять effective значения и bounded негативные probes, а не только binary version. До реализации этих gates local runner считается частично готовым; public disabled всегда.

Негативная матрица и public gates: [THREAT_MODEL](../architecture/THREAT_MODEL.md).
