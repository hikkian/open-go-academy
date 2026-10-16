# Operations plan — шаблон

## Capacity model

Опишите формулы с единицами: ingest bytes/s = events/s × mean bytes/event; retention bytes = stored bytes/s × retention seconds × replication/overhead factor; worker count = required events/s ÷ measured events/s per worker. Отделите измерения от предположений.

| Параметр | Значение/диапазон | Источник | Как проверить |
|---|---|---|---|
| average/peak events/s | | | |
| payload bytes | | | |
| retention | | | |
| retry multiplier | | | |

## SLI/SLO (предложение для согласования)

| SLI | Формула/окно | Цель | Исключения и источник |
|---|---|---|---|
| durable acceptance latency | | | |
| processing delay | | | |
| lost/duplicate business effects | | | |

Цель SLO является предложением, пока владелец не подтвердил её. Опишите error budget, saturation signals и алерты с действиями.

## Rollout and rollback

- Совместимость schema/event versions:
- Dark launch/canary и критерий продвижения:
- Stop conditions:
- Обратимый rollback и данные, которые rollback не отменит:

## Backup, restore, incident and on-call

- Какие данные/ключи копируются и кто имеет доступ:
- Restore sequence, измеряемый RTO и контроль RPO:
- Потеря/рост lag, credential compromise, poison queue — действия по минутам:
- Эскалация, коммуникация, владелец incident commander:
- Проверка runbook через game day:
