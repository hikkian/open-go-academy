# Middleware: request ID, безопасные логи и предел тела

## Общая работа вокруг разных handlers

Middleware (промежуточный обработчик) — handler, который оборачивает следующий handler. Он может выполнять общую работу до и после конкретного маршрута: добавить request ID, измерить длительность, записать ограниченный access log, применить body limit или установить заголовки. В Go это обычная функция, которая принимает `http.Handler` и возвращает `http.Handler`.

```text
Request → request ID → body limit → route handler → ResponseWriter
Response ←──────── access log ←──────── status capture ←
```

Обёртка, записывающая данные после `next.ServeHTTP`, увидит итоговый статус только если умеет отслеживать `WriteHeader`. Результат зависит от порядка: `A(B(handler))` сначала выполняет A до B, затем B до handler, затем завершает B и A. Порядок включения должен быть явно прочитан, а не угадан по именам функций.

Middleware не отменяет валидацию и авторизацию в конкретном API. Request ID связывает записи одного запроса; он не даёт права на ресурс и не является секретом. Генератор ID должен создавать уникальные случайные/непредсказуемые значения по подходящему системному источнику; учебная обёртка ниже принимает генератор извне, чтобы тестировать композицию детерминированно. Не используйте фиксированное значение как production ID.

## Что можно и нельзя логировать

Access log помогает узнать метод, результат, продолжительность и безопасный маршрут. Но запросы содержат credential в `Authorization`, cookie сессии, токены в query, персональные данные в body и иногда чувствительные значения в path. Безопасная стартовая политика — allowlist: логировать только явно нужные поля и никогда не писать целиком request headers/body. Маскирование на основе пары известных имён заголовков легко пропускает пользовательские секреты.

Не логируйте полный URL, если query может содержать секрет. Не записывайте response body. Если для диагностики нужно поле, сформулируйте разрешённый набор, ограничьте длину и рассмотрите токенизацию. Логи тоже являются чувствительными данными: задайте доступ, срок хранения и формат. Красивый JSON log сам по себе не безопасен.

Request ID — случайный или устойчиво уникальный идентификатор конкретного обмена. Прокси может прислать заголовок похожего имени. Нельзя без правил просто доверить клиентское значение: оно может содержать длинную строку, управляющие символы или обмануть корреляцию. Простое учебное решение генерирует собственный ID и перезаписывает заголовок; production контракт может принимать доверенную прокси-метку после строгой валидации и границы trust.

## Ограничения тела и паника

Middleware может оборачивать `r.Body` через `http.MaxBytesReader`, чтобы handler не прочёл больше лимита. Сам middleware не узнает о превышении, пока нижележащий handler не читает поток. Поэтому тот, кто декодирует JSON, обязан правильно сопоставить `*http.MaxBytesError` со статусом 413. Альтернатива — отдельная контролируемая предварительная проверка, но чтение должно оставаться ограниченным.

Panic recovery boundary может превратить панику в контролируемую ошибку запроса. Он не должен скрывать систематическую ошибку программиста или продолжать обработку частично записанного ответа. Если panic случилась после отправки заголовков, нельзя переписать уже наблюдавшийся статус. Не отправляйте panic value или stack trace клиенту. Сигнал для сервера безопасно собирается внутренней диагностикой; восстанавливать panic следует на узкой границе одного request, а не оставлять процесс в неизвестном состоянии.

## Пример 1. Внедрить генератор request ID

```go group=request-id-middleware mode=run
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

func WithRequestID(next http.Handler, newID func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newID()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

func main() {
	h := WithRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }), func() string { return "test-id-1" })
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/health", nil))
	fmt.Println(r.Code, r.Header().Get("X-Request-ID"))
}
```

Тестовый generator возвращает предсказуемый ID. В production генератор должен быть отдельной проверенной зависимостью с понятным поведением ошибки, а не константой. Request ID следует передавать в безопасный лог без использования в авторизации.

## Пример 2. Зафиксировать статус рядом с handler

```go group=status-capture mode=run
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status != 0 { return }
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func main() {
	r := httptest.NewRecorder()
	w := &statusWriter{ResponseWriter: r}
	(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) })).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/links?token=hidden", nil))
	fmt.Println(w.status, r.Code)
}
```

Полная совместимая обёртка должна учесть implicit `200` при первом `Write` и optional `Flusher`, `Hijacker`, `Pusher` интерфейсы там, где это нужно. Для простого JSON handler учебный код показывает механизм, а не готовый production middleware для всех протокольных расширений. Готовые библиотечные или тщательно проверенные wrappers могут быть уместны после понимания контракта.

## Пример 3. Читать только разрешённые поля лога

```go group=allowlist-log mode=run
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

func logRequest(dst io.Writer, method string, status int) {
	fmt.Fprintf(dst, "method=%s status=%d\n", method, status)
}

func main() {
	request := httptest.NewRequest(http.MethodGet, "/links?token=do-not-log", nil)
	recorder := httptest.NewRecorder()
	logRequest(recorder, request.Method, http.StatusOK)
	fmt.Print(recorder.Body.String())
}
```

Path, query и headers отсутствуют в формате. Это намеренный allowlist. Не добавляйте `r.URL.String()` «на время»: именно там может находиться token. Если маршруту нужна идентичность, используйте проверенное route name или безопасный resource type, а не произвольное клиентское значение.

## Порядок обёрток на примере

```text
handler := WithRequestID(AccessLog(BodyLimit(routes)), newID)
```

Внешний request-ID обёртывает весь запрос. Access log может использовать созданный заголовок или значение context, если тот безопасно установлен. Body limit обёртывает только выбранные routes. Перестановка меняет, что видит каждая обёртка. Например, recovery должна оборачивать код, который может паниковать; body limit должен сработать до чтения тела; авторизация обычно должна предшествовать дорогой бизнес-операции.

Композиция должна быть явной и тестируемой. Один маленький helper `Chain(handler, middleware...)` может снизить шум, но важно зафиксировать порядок применения в документации и тестах. Прочитайте цепочку снаружи внутрь, а затем выполните её мысленно обратно на response: внешний middleware завершится последним.

### Что должна сохранять ResponseWriter обёртка

ResponseWriter начинается как интерфейс записи headers и body, но стандартные writer в конкретном runtime могут также реализовать `http.Flusher`, `http.Hijacker`, `http.Pusher` или интерфейсы чтения из request body в отдельных протоколах. Middleware, которое обернуло writer только ради status, способно скрыть эти capabilities. Сервисный stream endpoint после такой обёртки неожиданно перестанет flush-ить heartbeat; websocket upgrade не увидит `Hijacker`.

Перед реализацией wrapper определите необходимые downstream возможности. Если endpoint обычный JSON и не использует streaming/upgrades, простой recorder может быть достаточен. Если система ими пользуется, preserve optional interfaces или применяйте адаптер, чья совместимость отдельно проверена. Нельзя объявлять учебный status recorder production-ready только потому, что он прошёл три теста.

### Panic и отмена запроса

Recovery внутри net/http может ограничить повреждение одним запросом, но ошибка требует внимания команды. Возвратите generic `500` только если response ещё не начат; никогда не отправляйте panic payload клиенту. В тесте поднимите panic до первой записи и проверьте статус, а затем отдельный сценарий записи перед panic покажет ограничение модели. Не подавляйте panic совсем без безопасного диагностического сигнала.

Request context отменяется при разрыве клиента, закрытии соединения или deadline. Middleware может проставить общий timeout, но он не заменяет timeout каждой сетевой зависимости. Не помещайте туда секреты и большую mutable state. В следующих занятиях передавайте `r.Context()` в хранилище, чтобы отмена запроса не запускала дорогую фоновую работу без пользователя.

### Измеряйте только пригодные данные

Для эксплуатационного access log часто достаточно route label, method, status, длительности и request ID. Raw path может включать email или другой объектный ID; замените его шаблоном маршрута (`/links/{alias}`), если он доступен. Поля должны быть ограничены, а типизированные значения отформатированы безопасно. Перед релизом прогоните тест с fixture секретами, затем проверьте формат для пустого, Unicode и длинного значения.

Если лог отправляется в JSON, escaping защищает формат записи от сломанной структуры, но не решает вопрос приватности. Если `Authorization` записан корректно экранированной JSON строкой, секрет всё равно раскрыт. Сначала решите, какие поля разрешено хранить; после этого выберите безопасный serializer и защиту хранилища логов.

## Типичные ошибки и диагностика

- **Фиксированный request ID в production.** Все запросы сольются в одну группу; ID не обязан быть секретом, но должен различать запросы.
- **Считать request ID полномочием.** Он коррелирует события и не удостоверяет личность или роль.
- **Логировать `r.URL.String()`.** Query может содержать token. Запускайте redaction-тест с очевидным секретным fixture.
- **Логировать headers по умолчанию.** Cookie и Authorization — credentials. Используйте allowlist.
- **Добавить MaxBytesReader и заявить, что превышение всегда уже отклонено.** Ошибка возникает при чтении ограниченного потока. Handler обязан её обработать.
- **Безусловно вызывать WriteHeader в wrapper.** Повторный header или скрытый implicit `200` повредит response. Отслеживайте первую запись.
- **Восстановить panic после отправки частичного JSON.** Статус и тело уже могли быть замечены. Не маскируйте состояние; перезапустите поток ответа понятным образом, если протокол это позволяет.
- **Оборачивать response writer и ломать streaming.** Сохранение `Flusher` и других optional интерфейсов может быть необходимым.

Для отладки создайте request с query `token=fixture-secret`, cookie `session=fixture-secret` и Authorization fixture, вызовите middleware и убедитесь, что ни одно значение не оказалось в log. Проверяйте отсутствие точного секрета, а не только наличие слова `redacted`. Затем проверьте, что status и ID дошли до клиента и не меняются по пути.

## Практика

- **Easy:** оберните handler генератором request ID. Для теста внедрите generator с последовательностью `id-1`, `id-2`; проверьте два запроса и response header.
- **Medium:** добавьте access log только с method и status. Подайте query token, Authorization и Cookie; ни одна fixture строка не должна появиться в log.
- **Hard:** примените жёсткий body limit до handler; oversized тело получает `413`, malformed — `400`, тело в пределах лимита проходит; проверьте порядок wrapper и не записывайте body/secret.

Условия должны сообщать порядок middleware и точные лимиты. Для deterministic тестов генератор ID и writer для логов инъецируются. Для oversized входа trusted driver строит body из selector, чтобы JSON cases остались меньше четырёх KiB. Успешные проверки измеряют статус, заголовки и redaction; точные пробелы лога не значимы, если контракт их не требует.

## Проверка понимания

1. Что именно делает middleware в модели `http.Handler`?
2. Почему композиция A(B(handler)) имеет порядок до/после?
3. Является ли request ID токеном доступа?
4. Почему allowlist для логов безопаснее списка «известных секретных header»?
5. Когда `MaxBytesReader` обнаруживает превышение?
6. Почему panic recovery не всегда может заменить уже начатый ответ?

Перед главой о SQL свяжите middleware с handler: первая общая граница для request ID и body limit; вторая — конкретная валидация; следующая — service/repository и хранилище. Повторите HTTP ответы и найдите места, где ошибка пользователя должна отличаться от внутреннего отказа.

## Мини-глоссарий

- **middleware** — handler-обёртка для общей работы вокруг следующего handler.
- **request ID** — идентификатор для корреляции одного обмена.
- **access log** — ограниченная запись о маршруте/статусе обработки.
- **redaction (редактирование)** — исключение или маскирование секретного значения.
- **body limit** — ограничение размера читаемого тела запроса.
- **allowlist** — список полей, которым явно разрешено попасть в log.
- **recovery boundary** — узкая граница восстановления после panic.
- **response wrapper** — обёртка, наблюдающая запись HTTP ответа.
