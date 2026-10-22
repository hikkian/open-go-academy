# Первый HTTP-сервер на net/http

## Что ты сделаешь

Ты напишешь три обработчика HTTP-запросов: `NewHealthHandler` ответит на проверку здоровья, `NewJSONEchoHandler` примет JSON и вернёт его обратно, `NewLinksCollectionHandler` отдаст фиксированный список. Каждый обработчик проверяет метод, путь и ответ так же, как это делает сервер, но без открытия порта.

## Handler — функция, отвечающая на один запрос

В `net/http` обработчик — любое значение с методом `ServeHTTP(w, r)`. Удобнее всего писать функцию с такой сигнатурой. Она получает `w` для записи ответа и `r` с запросом. Статус, заголовки и тело пишутся в определённом порядке: сначала заголовки, потом статус, потом тело:

```go
package main

import (
    "fmt"
    "net/http"
    "net/http/httptest"
)

func main() {
    h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusTeapot)
    })
    rec := httptest.NewRecorder()
    h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
    fmt.Println(rec.Code)
}
```

Вывод: `418`. `httptest.NewRecorder` записывает ответ в память, а `httptest.NewRequest` создаёт запрос. Так обработчик проверяется без сети.

> **Важно.** Заголовки нужно задать до `WriteHeader` или первой записи тела. После этого их уже не изменить.

## Метод и путь

`http.ServeMux` направляет запрос по пути. Путь `/health` без завершающего слэша совпадает точно: `/health/` и `/missing` получат 404 автоматически. Метод проверяет сам обработчик. Если метод неверный, нужно вернуть 405 и заголовок `Allow` со списком разрешённых методов:

```go
package main

import (
    "fmt"
    "net/http"
    "net/http/httptest"
)

func health(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.Header().Set("Allow", http.MethodGet)
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Write([]byte(`{"status":"ok"}` + "\n"))
}

func main() {
    rec := httptest.NewRecorder()
    health(rec, httptest.NewRequest(http.MethodPost, "/health", nil))
    fmt.Println(rec.Code, rec.Header().Get("Allow"))
}
```

Вывод: `405 GET`. `HEAD` тоже не `GET`: по контракту задания он отвергается с 405, даже если на `GET` сервер отвечает телом.

> **Типичная ошибка.** Ставить `Allow` после `http.Error`. Заголовок уже ушёл клиенту, и в ответе его не будет.

## JSON на входе: строго

Тело запроса читается потоком. `json.Decoder` с `DisallowUnknownFields` отвергает поля, которых нет в структуре, и незакрытый JSON. Пустое значение поля тоже отвергается, если контракт этого требует:

```go
package main

import (
    "encoding/json"
    "fmt"
    "strings"
)

type request struct {
    Message string `json:"message"`
}

func main() {
    dec := json.NewDecoder(strings.NewReader(`{"message":"x","extra":1}`))
    dec.DisallowUnknownFields()
    var req request
    fmt.Println(dec.Decode(&req) != nil)
}
```

Вывод: `true`. Лишнее поле `extra` — ошибка декодирования, а не молчаливое игнорирование.

## Если что-то пошло не так

- `undefined: http` → не добавлен импорт `"net/http"`.
- Клиент получает 200 с пустым телом → обработчик не записал ответ; `w.Write` или `json.NewEncoder(w).Encode` должны вызываться.
- `http: superfluous response.WriteHeader call` → статус записан дважды: оставь один `WriteHeader` или `http.Error`.
- Неизвестный путь даёт 405 вместо 404 → метод проверяется до пути, а не внутри обработчика нужного пути.

## Практика

Три задания. Обработчики проверяются на запросах из блока «Ожидаемый вывод»; закрытые примеры видны только в «Проверить». Сервер не запускается: проверка работает через записи в памяти.

### Задание 1 · Ответь на health запрос

**Цель.** `GET /health` отвечает 200 и JSON `{"status":"ok"}`. Другой метод на этом пути — 405 с `Allow: GET`. Любой другой путь — 404.

**Ожидаемый вывод** — запросы `GET /health` и `POST /health`. Каждый ответ печатается как объект с полями `Status`, `Body`, `ContentType`, `Location`, `Allow`:

```text
{"Status":200,"Body":"{\"status\":\"ok\"}\n","ContentType":"application/json; charset=utf-8","Location":"","Allow":""}
{"Status":405,"Body":"method not allowed\n","ContentType":"text/plain; charset=utf-8","Location":"","Allow":"GET"}
```

**Шаги.**
1. Выбери задание «Ответь на health запрос»: в файле есть `NewHealthHandler` с `panic("TODO")`. Импорт `"net/http"` уже есть.
2. Создай маршрутизатор: `mux := http.NewServeMux()`.
3. Зарегистрируй путь `/health` с функцией, которая проверяет метод.
4. Для `GET` задай `Content-Type` и запиши JSON `{"status":"ok"}` с переводом строки.
5. Для другого метода задай `Allow` и ответь через `http.Error` со статусом 405.
6. Верни `mux`, нажми «Запустить» и сравни с блоком выше.
7. Нажми «Проверить».

**Как проверить себя.** Два объекта ответа; все три критерия зелёные.

**Если не принято.**
- Для `POST` вышло 200 → метод не проверяется.
- Для `HEAD` вышло 200 → по контракту `HEAD` тоже отвергается с 405.
- Путь `/health/` даёт 200 → такой путь должен давать 404: регистрируется только точный путь.

> **Подсказка.** Проверка метода — первая строка функции, а не отдельный маршрут.

### Задание 2 · Проверь JSON echo handler

**Цель.** `POST /echo` принимает JSON с полем `message` и непустым значением, возвращает его как JSON с 200. Повреждённый JSON, пустое сообщение и лишнее поле — 400 с текстом `invalid request`. Другой метод — 405 с `Allow: POST`.

**Ожидаемый вывод** — валидный запрос с `message` равным `hello`, затем `GET /echo`:

```text
{"Status":200,"Body":"{\"message\":\"hello\"}\n","ContentType":"application/json; charset=utf-8","Location":"","Allow":""}
{"Status":405,"Body":"method not allowed\n","ContentType":"text/plain; charset=utf-8","Location":"","Allow":"POST"}
```

**Шаги.**
1. Выбери задание «Проверь JSON echo handler»: в файле есть `NewJSONEchoHandler` с `panic("TODO")`.
2. Создай маршрутизатор и зарегистрируй путь `/echo`.
3. Если метод не `POST`, задай `Allow: POST` и ответь 405.
4. Декодируй тело через `json.NewDecoder(r.Body)` с `DisallowUnknownFields`.
5. Если декодирование не удалось или `message` пустое, ответь 400 через `http.Error` с текстом `invalid request`.
6. Иначе задай `Content-Type` и запиши объект `{"message": ...}` через `json.NewEncoder(w)`.
7. Нажми «Запустить» и сравни с блоком выше.
8. Нажми «Проверить».

**Как проверить себя.** Два объекта ответа; все три критерия зелёные.

**Если не принято.**
- Лишнее поле принято → декодер нужно настроить через `DisallowUnknownFields`.
- Пустое сообщение даёт 200 → проверка пустоты стоит после декодирования.
- Повреждённый JSON даёт панику → ошибку декодирования надо обработать, а не игнорировать.

> **Подсказка.** Один `json.Decoder` на тело запроса; структура с одним полем `Message` и тегом `json:"message"`.

### Задание 3 · Отдай согласованную коллекцию

**Цель.** `GET /links` отвечает 200 и фиксированным JSON-списком ссылок. Другой метод на этом пути — 405 с `Allow: GET`. Прочие пути — 404. Хранилище не используется.

**Ожидаемый вывод** — `GET /links` и `POST /links`:

```text
{"Status":200,"Body":"[{\"alias\":\"go\",\"target\":\"https://go.dev/\"}]\n","ContentType":"application/json; charset=utf-8","Location":"","Allow":""}
{"Status":405,"Body":"method not allowed\n","ContentType":"text/plain; charset=utf-8","Location":"","Allow":"GET"}
```

**Шаги.**
1. Выбери задание «Отдай согласованную коллекцию»: в файле есть `NewLinksCollectionHandler` с `panic("TODO")`.
2. Создай маршрутизатор и зарегистрируй путь `/links`.
3. Если метод не `GET`, задай `Allow: GET` и ответь 405.
4. Для `GET` задай `Content-Type` и запиши фиксированный список одной записи: `alias` со значением `go` и `target` со значением `https://go.dev/`.
5. Верни маршрутизатор, нажми «Запустить» и сравни с блоком выше.
6. Нажми «Проверить».

**Как проверить себя.** Два объекта ответа; все три критерия зелёные.

**Если не принято.**
- `HEAD /links` даёт 200 → проверка метода должна стоять до ответа.
- `/links/x` даёт 200 → путь зарегистрирован с завершающим слэшем; нужен точный `/links`.
- Список записан вручную с пробелами в JSON → тело должно совпасть с кодировкой без лишних пробелов.

> **Подсказка.** Фиксированный ответ можно записать строкой; `json.NewEncoder` даёт тот же вид.

## Проверь себя

- Что такое handler и какие два аргумента он получает?
- Почему статус и заголовки надо задать до тела ответа?
- Почему `http.ServeMux` даёт 404 для `/health/`, а не для `/health`?
- Что делает `DisallowUnknownFields`?
- Почему `HEAD` тоже отвергается при контракте только для `GET`?
- Как проверить обработчик без открытия порта?

## Итог

- Handler отвечает на один запрос; статус и заголовки записываются до тела.
- Метод проверяется в обработчике и возвращает 405 с `Allow`; путь направляет `ServeMux`.
- Входной JSON читается строго: неизвестные поля и пустые значения отвергаются с 400.
