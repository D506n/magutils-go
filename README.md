# MagUtils Go

Перенос полезных утилит из [magutils](https://github.com/D506n/magutils) (Python) в Go.

## Статус переноса

| Модуль | Статус | Описание |
|--------|--------|----------|
| [`id`](id/id.go) | ✅ Готово | Генерация коротких уникальных ID (nanoid) |
| [`logging/handlers`](logging/handlers) | ✅ Готово | Кастомные `slog.Handler`: цветной консольный и JSON |
| [`star`](star/star.go) | ✅ Готово | Выполнение Starlark-скриптов в изолированном контексте |
| `env` | ✅ Готово | Загрузка конфигурации из `.env` файлов и переменных окружения |

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

## Тестирование

```bash
# Все тесты
go test ./...

# Тесты конкретного пакета
go test ./star/...
go test ./id/...
go test ./logging/handlers/...

# С бенчмарками
go test -bench=. ./id/...
```

---

## Лицензия

MIT