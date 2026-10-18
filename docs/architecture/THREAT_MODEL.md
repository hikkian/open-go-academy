# Модель угроз execution и credentials

Дата модели: 2026-10-15; evidence обновлено 2026-10-18. Статус: модель угроз остается IN PROGRESS; некоторые bounded probes выполнены на текущей Linux/rootless Podman/cgroup v2 конфигурации, полный набор S01–S12 не закрыт. Источники механизмов и фактические probes: [SANDBOX_SECURITY](../research/SANDBOX_SECURITY.md); модель доверия: [CREDENTIALS](../research/CREDENTIALS.md).

## Область и границы

Режим по умолчанию — браузерное приложение на loopback для одного владельца установки; public runner выключен. Ученик контролирует source, file names, stdout/stderr и время/последовательность HTTP requests. Вредоносная внешняя страница также может обращаться к localhost: loopback не заменяет browser-origin защиту. Администратор ОС и пользователь, меняющий сервер/БД на своей установке, вне гарантий её аттестации.

Активы: host-файлы и секреты; доступность компьютера; DB прогресса; скрытые критерии; issuer keys; личные данные; достоверность rank/credential. Kernel, закреплённый toolchain и политика worker — trusted computing base. Kernel escape, аппаратные side channels и скомпрометированный администратор не исключаются namespaces.

Поток: Browser → API (валидация/auth/origin/лимиты) → bounded queue → отдельный worker → per-job isolated compiler/test/binary → bounded result → серверная grade transaction → DB. Sandbox output является недоверенным текстом, exit code/stdout ученика не являются полномочием присвоить rank. Issuer получает серверную evidence, ключ не пересекает worker boundary.

## Негативные gates перед допуском local execution

| ID | Воспроизводимый bounded probe | Требуемый результат и evidence |
|---|---|---|
| S01 | Программа читает специально созданный host sentinel вне mounts; пишет туда; читает `/proc/1/root`, `/proc/1/environ`, socket Docker/Podman и путь issuer key | Sentinel/секреты не доступны; host unchanged; sandbox mount list сохранён |
| S02 | TCP/UDP к локальному тестовому listener, host loopback, внешнему sentinel и DNS; обращения к unix socket вне sandbox | Ни одного соединения на host listener; network namespace и запреты документированы |
| S03 | Создание namespace/mount, ptrace другого job, raw socket, привилегированный syscall | Denied; отсутствие capabilities; loaded seccomp policy проверена |
| S04 | Бесконечный loop | Timeout в пределах бюджета+ограниченного cleanup; `populated=0`, нет оставшихся descendants |
| S05 | Bounded memory allocator и запуск дочерних процессов в ограниченной sandbox | Per-job memory/PID limits подтверждены actual cgroup counters; host/API работоспособны |
| S06 | Bounded многофайловая запись и flood обоих output streams | Scratch quota и общий output cap; predictable result, reader не deadlock; cleanup |
| S07 | Parent завершился, ребёнок удерживает stdout; cancel в queue/run; worker crash/restart | Readers завершаются, группа убита, слот освобождён ровно один раз, stale jobs удалены |
| S08 | `../`, absolute path, duplicate names, NUL, symlink/replacement попытки, oversized body | Reject до исполнения; запись только в приватный job root; ошибка не раскрывает host paths |
| S09 | Два job с одинаковыми filename; overload bounded queue | Нет смешения outputs/cache/workspaces; concurrency cap, отказ/429 без накопления памяти |
| S10 | Ученик печатает «PASS», подделывает JSON result, меняет tests/package init/test helper | Result парсится доверенным grader; студент не получает полномочий менять evidence/rank |
| S11 | Sandbox binary отсутствует, namespace запрещён, cgroup не delegated, policy invalid | Run/submit unavailable; ноль пользовательских processes на host; нет unsafe fallback |
| S12 | Foreign Origin/Host, cross-site POST/fetch, DNS rebinding, forged client rank | Request rejected; exact Origin/Host allowlist, CSRF/session protection; server-authoritative grade |

**S05: локальное подтверждение 2026-10-18.** `make sandbox-test` завершился exit 0. В текущей rootless Podman/cgroup v2 конфигурации runner memory fixture был остановлен ядром с exit 137 и `memory.events: oom_kill=1`; PID fixture получил `EAGAIN` (`resource temporarily unavailable`), очистил все 33 дочерних процесса и завершился с exit 0 при `pids.events: max=3`. Проверка воспроизводится `scripts/m12_sandbox_limit_probe.py`; итог записан в `docs/validation/m12-sandbox-cgroup-20261018.txt`. Это подтверждает текущую локальную конфигурацию и эти два сценария, не является доказательством для другого ядра/runtime, публичной multi-tenant модели или всей S01–S12 матрицы.

**S01/S02/S03/S06/S07/S09/S11: bounded local probes 2026-10-18.** Rootless `make sandbox-test` прошел на current tree (`artifacts/m12-final-batch-sandbox-test-20261018.log`): обе output streams вместе достигли общего лимита 32768 bytes и вернули `output_limit`; запись в разные файлы заняла 189 отдельных 1 MiB файлов и получила `ENOSPC`; parent-exit/child-inherited-pipes probe завершился за 3.12s с точной scoped-container cleanup проверкой. Same-filename concurrent job отказан как busy, первый job вернул только собственный stdout. Fake-runtime S11 fixtures отвергли cgroup v1 и отсутствующую delegation без user-job запуска; только scoped preflight запускался и очищался. S03 syscall fixture подтвердил `unshare(CLONE_NEWNS)`, mount, raw socket и ptrace container PID 1 denied; `CapEff=0`, `Seccomp=2`.

Host-boundary probe (`scripts/m12_sandbox_access_probe.py`) показал denied read/write на временный sentinel и issuer-key fixture, включая путь через `/proc/1/root`; fixture остались unchanged, sentinel был 0644, issuer-key test fixture 0600. Host-only environment marker отсутствовал в container `/proc/1/environ`, standard Docker/Podman socket paths отсутствовали. Временные host TCP/UDP/unix listeners получили 0 соединений/датаграмм; TCP и Unix connect завершились отказом. UDP `Write` вернул успех, но listener datagram не получил — это наблюдение, не доказательство того, что syscall отказал.

Остаются: `cgroup.events populated=0` literal для S04; DNS/external-network behavior; snapshot/сопоставление полного host mount list и host PID-1 (тест сверил только namespace container PID-1); scope иных host kernels. S04 timeout и отсутствие scoped container подтверждены, но не literal counter. S02 не выполняет внешний network/DNS probe. Подробности: `docs/validation/m12-host-boundary-20261018.txt` и `docs/validation/m12-api-boundary-probes-20261018.txt`.

Для S05/S06 нельзя сначала создавать необузданный host fork/memory/disk bomb. Harness предварительно создаёт и проверяет ограниченную группу, имеет внешний watchdog и host sentinel. Все tests сохраняют команды, лимиты, версии, exit status, elapsed time и cleanup evidence. Успешный hello-world не заменяет матрицу.

## Hidden tests и correctness

Go tests, скомпилированные с пользовательским package, находятся в его execution domain: исходники/логика/данные теста могут стать доступны злонамеренному коду. Не обещать их секретность. Держать criteria и финальный verdict вне sandbox; не верить произвольному `go test` stdout как основанию credentials. Для adversarial assessments нужен отдельный trusted harness/протокол, непредсказуемые cases, immutable evidence и независимое ревью. Локальный владелец исходников всё равно может изучить проверки; это не прокторинг.

## Public deployment: блокирующие требования

1. Отдельный непривилегированный execution service на выделенных workers; API/DB/keys на других trust boundaries. Прямой runtime socket/privileged flags пользователю недоступны.
2. Проверенный gVisor либо microVM backend, patched kernel/runtime/toolchain, минимальные pinned images, seccomp/LSM policy; исключить автоматический переход на обычный OCI/process при ошибке.
3. Полная negative matrix на реальном deployment, независимый security review и закрытые high/critical findings. Resource isolation проверять между tenants; quotas на пользователя/tenant, admission control, rate limits, abuse monitoring и cleanup после падений.
4. Auth/session/CSRF/IDOR, TLS, exact origin policy, tenant isolation и private-by-default данные; Internet и metadata service закрыты; политика для dependencies отдельная от запуска.
5. Credential issuance из immutable серверной exam evidence; idempotency, atomic transaction, replay/race tests, key rotation/revocation и incident runbook.

Остаточный риск local: общий kernel и UID владельца, возможные runtime/kernel уязвимости, side channels, DoS в пределах разрешённых лимитов. Для hostile public workloads local профиль не допускается даже при зелёных локальных tests. Публикация требует отдельного разрешения владельца проекта и выполнения gates; этот документ её не разрешает.

## Evidence M3–M6 (2026-10-15)

Local integration matrix12cases PASS118.231s, actual API restart и runnerSIGKILL recovery PASS, core browser10passed/2mobile runtime duplicates skipped, P1 expanded15cases+unit PASS. Commands/logs and exact limits: ../VALIDATION_REPORT.md. S01/S02/S03 проверены sentinel/RO root/TCP/mount plus preflightSeccomp/caps; не заявляются exhaustive ptrace/UDP/DNS/cross-tenant checks. Public gates по-прежнему blocked. exec cache/work являются частью одной per-job filesystem/cgroup boundary; no host mounts, per-job caches, no shared student state. Student test stdout не issuer authority; manual-review flags остаются true, ranks/credentials не выданы.
