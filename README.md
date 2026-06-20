# MagUtils Go

Перенос полезных утилит из [magutils](https://github.com/D506n/magutils) (Python) в Go.

## Статус переноса

| Модуль | Статус | Описание |
|--------|--------|----------|
| [`id`](id/id.go) | ✅ Готово | Генерация коротких уникальных ID (nanoid) |
| [`json-path`](json-path) | ✅ Готово | DSL для навигации и манипуляции вложенными JSON-структурами |
| [`logging/handlers`](logging/handlers) | ✅ Готово | Кастомные `slog.Handler`: цветной консольный и JSON |
| [`star`](star/star.go) | ✅ Готово | Выполнение Starlark-скриптов в изолированном контексте |
| [`env`](env/env.go) | ✅ Готово | Загрузка конфигурации из `.env` файлов и переменных окружения |

---


## id

Пакет для генерации коротких уникальных идентификаторов на основе алгоритма [NanoID](https://zelark.github.io/nano-id-cc/). Использует `crypto/rand` через библиотеку `go-nanoid/v2`.

Алфавит по умолчанию: `A-Za-z0-9` (62 символа). Длина по умолчанию: 15. Для вероятности коллизии 1% нужно сгенерировать ~3 триллиона ID.

### Установка

```go
import "github.com/D506n/magutils-go/id"
```

### Использование

```go
// Генерация с настройками по умолчанию
id1, err := id.Gen()
// id1: "9VkeORlHnV1pMjA"

// MustGen — паникует при ошибке (удобно для инициализации)
id2 := id.MustGen()

// Кастомный алфавит и длина
id.SetAlphabet("0123456789abcdef")
id.SetSize(10)
id3, _ := id.Gen()
// id3: "a3f2c9b1e7"

// Валидация
err := id.IsValid(id3)
// err == nil — ID валиден
```

---

## logging/handlers

Кастомные [`slog.Handler`](https://pkg.go.dev/log/slog#Handler) для форматирования логов. Аналог Python-модуля `magutils.logging.formatters`.

### ColorHandler

Цветной вывод в консоль с ANSI-кодами. Аналог `ColoredConsoleFormatter`.

```go
import (
    "log/slog"
    "os"
    "github.com/D506n/magutils-go/logging/handlers"
)

h := handlers.NewColorHandler(os.Stderr)
logger := slog.New(h)

logger.Info("сервер запущен", "port", 8080)
logger.Warn("дисковое пространство заканчивается", "free_gb", 1.5)
logger.Error("ошибка подключения к БД", "retry", 3)
```

Формат вывода: `[LEVEL|time|file:line] message`

Цвета по уровням:
- `DEBUG` — Cyan
- `INFO` — Green
- `WARN` — Yellow
- `ERROR` — Red

### JSONHandler

Вывод логов в JSON. Аналог `JsonFormatter`.

```go
h := handlers.NewJSONHandler(os.Stdout)
logger := slog.New(h)

logger.Info("запрос обработан",
    "method", "GET",
    "path", "/api/users",
    "status", 200,
)
```

Формат вывода:
```json
{
    "time": "2026-06-20T00:00:00Z",
    "level": "INFO",
    "message": "запрос обработан",
    "source": {"file": "/path/to/main.go", "line": 42},
    "method": "GET",
    "path": "/api/users",
    "status": 200
}
```

---

## star

Пакет для выполнения скриптов на языке [Starlark](https://github.com/google/starlark-go) (диалект Python, используемый в Bazel/Buck) в изолированном контексте. Аналог Python-модуля `magutils.star.starlark`.

### Установка

```go
import "github.com/D506n/magutils-go/star"
```

### Использование

```go
runner := star.NewRunner()

// Скрипт — тело функции process(input)
script := `
print("обработка данных")
match = re.search(r'\\d+', input['text'])
return {'match': match}
`

// Входные данные
data := starlark.NewDict(1)
data.SetKey(starlark.String("text"), starlark.String("abc123def"))

// Выполнение
res := runner.Run(script, data)

if res.Success {
    fmt.Println("Результат:", res.Value)  // {"match": "123"}
    fmt.Println("Print:", res.Prints)     // ["обработка данных"]
} else {
    fmt.Println("Ошибка:", res.Error)
}
```

### Встроенные функции

В скриптах доступны:

- **`print(*msgs)`** — вывод сообщений (сохраняется в `Result.Prints`)
- **`re`** — структура с методами:
  - `re.findall(pattern, text)` — все совпадения regex
  - `re.search(pattern, text, group=0)` — первое совпадение с выбором группы
- **`time`** — структура с методами:
  - `time.now()` — текущее время в секундах с эпохи (unix timestamp)
  - `time.start` — время старта скрипта
  - `time.elapsed()` — секунд с начала выполнения
  - `time.sleep(seconds)` — пауза (принимает int или float)
- **`struct(field=value, ...)`** — создание структуры с полями

### Кастомизация

```go
// Кастомный setup (дополнительные функции)
customSetup := star.Setup(`
def double(x):
    return x * 2
`)

// Кастомный wrapper (изменение логики обёртки)
customWrapper := star.Wrapper(`
{setup}
def process(input):
{script}
def main(inp):
    result = process(inp)
    return {'custom': result}
results = main(input)
`)

runner := star.NewRunner(
    star.WithSetup(customSetup),
    star.WithWrapper(customWrapper),
)
```

### Расширение Go-функциями (WithBuiltin)

Через опцию `WithBuiltin` можно добавить любую Go-функцию в Starlark-окружение. Сигнатура функции:

```go
func myFunc(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)
```

Для разбора аргументов используй `starlark.UnpackArgs`:

```go
func httpGet(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
    var url string
    if err := starlark.UnpackArgs("http_get", args, kwargs, "url", &url); err != nil {
        return nil, err
    }
    // ... делаем HTTP-запрос
    return starlark.String(body), nil
}

runner := star.NewRunner(
    star.WithBuiltin("http_get", starlark.NewBuiltin("http_get", httpGet)),
)
```

Для группировки функций в структуру (как `http.get`, `http.post`) используй `starlarkstruct`:

```go
httpStruct := starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
    "get":  starlark.NewBuiltin("http_get", httpGet),
    "post": starlark.NewBuiltin("http_post", httpPost),
})

runner := star.NewRunner(
    star.WithBuiltin("http", httpStruct),
)
```

В Starlark это будет работать как:

```starlark
data = http.get("https://api.example.com/data")
resp = http.post("https://api.example.com/submit", body)
```

### Как это работает

Пользовательский скрипт автоматически оборачивается в шаблон:

```starlark
{setup}

def process(input):
{script}

def main(inp):
    result = process(inp)
    if result == None:
        return {}
    if type(result) not in ('dict', 'list'):
        result = {'result': result}
    return result

results = main(input)
```

- Пользователь пишет только тело `return 'Hello starlark!'`
- Вывод: `{'result': 'Hello starlark'}`

### Отличия от Python-версии

| Аспект | Python | Go |
|--------|--------|----|
| Движок | `starlark-pyo3` (Rust FFI) | `go.starlark.net` (чистый Go) |
| Пул контекстов | `asyncio.Queue` | Не нужен (создание Thread дёшево) |
| Singleton | `Runner.inst()` | Не нужен (просто `NewRunner()`) |
| Асинхронность | `async/await` + `aio.to_thread` | Синхронный (горутины) |

---

## json-path

Пакет для навигации и манипуляции вложенными JSON-структурами (`map[string]any` / `[]any`) через строковые пути. Аналог Python-модуля `magutils.json_path`.

### Установка

```go
import "github.com/D506n/magutils-go/json-path"
```

### Синтаксис пути

| Пример | Сегменты | Описание |
|--------|----------|----------|
| `a.b.c` | `Key → Key → Key` | Доступ по ключам словаря |
| `items.0.id` | `Key → Index → Key` | Доступ по индексу списка |
| `items.-1.id` | `Key → Index → Key` | Отрицательный индекс (с конца) |
| `items.*.id` | `Key → Wildcard → Key` | Обход всех элементов списка |
| `list.!a` | `Key → Append` | Добавление в конец списка |

### Использование

```go
data := map[string]any{
    "user": map[string]any{
        "name": "Alice",
        "tags": []any{"admin", "editor"},
    },
}

// GetByPath — чтение
vals, _ := jsonpath.GetByPath("user.name", data)
fmt.Println(vals) // ["Alice"]

vals, _ = jsonpath.GetByPath("user.tags.*", data)
fmt.Println(vals) // ["admin", "editor"]

// С значением по умолчанию
vals, _ = jsonpath.GetByPath("user.missing.key", data,
    jsonpath.WithDefault("fallback"))
fmt.Println(vals) // ["fallback"]

// SetByPath — запись
jsonpath.SetByPath("user.name", data, "Bob")
// data["user"]["name"] == "Bob"

// Автосоздание промежуточных структур
jsonpath.SetByPath("a.b.c", data, 42)
// data["a"]["b"]["c"] == 42

// Append в конец списка
jsonpath.SetByPath("user.tags.!a", data, "moderator")
// data["user"]["tags"] == ["admin", "editor", "moderator"]

// DelByPath — удаление
jsonpath.DelByPath("user.tags.0", data)
// data["user"]["tags"] == ["editor", "moderator"]

// Wildcard-удаление
jsonpath.DelByPath("user.tags.*", data)
// data["user"]["tags"] == []
```

### DeepMerge — глубокое слияние

```go
base := map[string]any{"a": 1, "b": map[string]any{"c": 2}}
patch := map[string]any{"b": map[string]any{"d": 3}, "e": 4}

merged := jsonpath.DeepMerge(base, patch)
// merged == {"a": 1, "b": {"c": 2, "d": 3}, "e": 4}
// base — не изменён

// Без копирования (мутирует base)
merged2 := jsonpath.DeepMerge(base, patch, false)
// base == merged2
```

### DeepCopy — глубокое копирование

```go
original := map[string]any{"a": []any{1, 2, 3}}
copied := jsonpath.DeepCopy(original).(map[string]any)

copied["a"].([]any)[0] = 99
// original["a"][0] == 1 — не изменился
```

### Rebuild — трансформация данных

```go
data := map[string]any{
    "items": []any{
        map[string]any{"id": 10, "val": "x"},
        map[string]any{"id": 20, "val": "y"},
    },
}

// Простое перекладывание
result, _ := jsonpath.Rebuild(data, "items.0.id->target")
// result == {"target": 10}

// Несколько трансформаций
result, _ = jsonpath.Rebuild(data,
    "items.*.id-> *.id",
    "items.*.val-> *.value",
)
// result == [{"id": 10, "value": "x"}, {"id": 20, "value": "y"}]
```

### Format — шаблонизация строк

```go
data := map[string]any{"name": "Alice", "role": "admin"}
text, _ := jsonpath.Format("User {name} has role {role}", data)
// text == "User Alice has role admin"
```

### DictToPaths — генерация путей из структуры

```go
data := map[string]any{
    "a": map[string]any{"b": 1, "c": 2},
    "d": []any{map[string]any{"e": 3}},
}

paths, _ := jsonpath.DictToPaths(data)
// paths == ["a.b", "a.c", "d.*.e"]

paths, _ = jsonpath.DictToPaths(data, "strict")
// paths == ["a.b", "a.c", "d.0.e"]
```

---

## Тестирование

```bash
# Все тесты
go test ./...

# Тесты конкретного пакета
go test ./star/...
go test ./id/...
go test ./json-path/...
go test ./logging/handlers/...

```

---

## Лицензия

MIT