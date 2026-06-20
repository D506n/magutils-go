// Package star предоставляет базовый Runner для выполнения Starlark-скриптов.
//
// Аналог Python-модуля magutils/star/starlark.py
package star

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// ========================================
// Типы для конфигурации
// ========================================

// Setup — Starlark-код, определяющий глобальные функции и переменные.
// Аналог DEFAULT_SETUP в Python.
type Setup string

// Wrapper — шаблон обёртки скрипта.
// {script} заменяется на тело пользовательского скрипта.
// {setup} заменяется на Setup-код.
// Аналог DEFAULT_WRAPPER в Python.
type Wrapper string

// Option — функциональная опция для Runner.
type Option func(*Runner)

// ========================================
// Значения по умолчанию
// ========================================

const (
	// DefaultSetup — базовый набор функций, доступных в Starlark-скрипте.
	// Аналог DEFAULT_SETUP из Python.
	DefaultSetup = Setup(`
def star_elapsed():
    return time_now() - _star_start

re = struct(
    findall = star_re_findall,
    search = star_re_search,
)

time = struct(
    now = time_now,
    start = _star_start,
    elapsed = star_elapsed,
    sleep = time_sleep,
)
`)

	// DefaultWrapper — стандартная обёртка скрипта.
	// Аналог DEFAULT_WRAPPER из Python.
	DefaultWrapper = Wrapper(`
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
`)
)

// ========================================
// Result — результат выполнения скрипта
// ========================================

// Result содержит результат выполнения Starlark-скрипта.
// Аналог StarResult из Python.
type Result struct {
	Value   starlark.Value // результат (из globals["results"])
	Prints  []string       // всё, что напечатано через print()
	Error   error          // ошибка выполнения, если была
	Success bool           // флаг успеха
}

// ========================================
// Runner — базовый раннер Starlark-скриптов
// ========================================

// Runner выполняет Starlark-скрипты с заданным setup и wrapper.
// Аналог Runner из Python, но без пула контекстов (в Go он не нужен).
type Runner struct {
	mu       sync.Mutex
	setup    Setup
	wrapper  Wrapper
	builtins starlark.StringDict // предзагруженные builtin-функции
	cache    sync.Map            // map[string]string — кеш обёрнутых скриптов
}

// NewRunner создаёт Runner с настройками по умолчанию.
func NewRunner(opts ...Option) *Runner {
	r := &Runner{
		setup:   DefaultSetup,
		wrapper: DefaultWrapper,
	}
	for _, opt := range opts {
		opt(r)
	}
	r.initBuiltins()
	return r
}

// WithSetup заменяет setup-код.
func WithSetup(s Setup) Option {
	return func(r *Runner) {
		r.setup = s
	}
}

// WithWrapper заменяет wrapper-код.
func WithWrapper(w Wrapper) Option {
	return func(r *Runner) {
		r.wrapper = w
	}
}

// initBuiltins инициализирует встроенные функции Starlark.
// Аналог BaseCTX.setup() в Python.
func (r *Runner) initBuiltins() {
	r.builtins = starlark.StringDict{
		"struct": starlark.NewBuiltin("struct", structBuiltin),

		// re
		"star_re_findall": starlark.NewBuiltin("star_re_findall", reFindall),
		"star_re_search":  starlark.NewBuiltin("star_re_search", reSearch),

		// time
		"time_now":   starlark.NewBuiltin("time_now", timeNow),
		"time_sleep": starlark.NewBuiltin("time_sleep", timeSleep),

		// print — кастомный, форматирует float как в Python
		"print": starlark.NewBuiltin("print", starPrint),
	}
}

// buildWrapped собирает финальный Starlark-скрипт из setup + wrapper + user script.
// Аналог wrap_script() + build_wrapper() в Python.
func (r *Runner) buildWrapped(script string) string {
	// Индентируем пользовательский скрипт (3 пробела, как в Python)
	indented := indent(script, "   ")

	// Собираем wrapper: заменяем {setup} и {script}
	wrapper := string(r.wrapper)
	wrapper = strings.ReplaceAll(wrapper, "{setup}", string(r.setup))
	wrapper = strings.ReplaceAll(wrapper, "{script}", indented)

	return wrapper
}

// wrapScript возвращает обёрнутый скрипт, используя кеш.
// Аналог wrap_script() с @lru_cache в Python.
func (r *Runner) wrapScript(script string) string {
	if cached, ok := r.cache.Load(script); ok {
		return cached.(string)
	}
	wrapped := r.buildWrapped(script)
	r.cache.Store(script, wrapped)
	return wrapped
}

// Run выполняет Starlark-скрипт с переданными данными.
// Аналог Runner._run() в Python.
//
// script — пользовательский Starlark-код (тело функции process).
// data — входные данные (будет доступно как переменная input).
func (r *Runner) Run(script string, data starlark.Value) *Result {
	res := &Result{}

	// 1. Оборачиваем скрипт
	wrapped := r.wrapScript(script)

	// 2. Создаём Thread
	var prints []string
	thread := &starlark.Thread{
		Name: "star",
		Print: func(_ *starlark.Thread, msg string) {
			prints = append(prints, msg)
		},
	}

	// 3. Готовим глобалы: builtins + input + _star_start
	globals := make(starlark.StringDict)
	for k, v := range r.builtins {
		globals[k] = v
	}
	globals["input"] = data
	globals["_star_start"] = starlark.Float(float64(time.Now().UnixMilli()) / 1000.0)

	// 4. Выполняем
	globalsOut, err := starlark.ExecFile(thread, "main.star", wrapped, globals)
	if err != nil {
		res.Error = err
		res.Prints = prints
		return res
	}

	// 5. Забираем результат
	res.Success = true
	res.Value = globalsOut["results"]
	res.Prints = prints
	return res
}

// starPrint — кастомный print, который форматирует float как в Python.
func starPrint(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var parts []string
	for _, arg := range args {
		parts = append(parts, formatValue(arg))
	}
	msg := strings.Join(parts, " ")
	// Используем Print из thread, если есть
	if thread.Print != nil {
		thread.Print(thread, msg)
	}
	return starlark.None, nil
}

// formatValue форматирует starlark.Value как Python print().
// starlark.Float.String() использует %g, который для больших чисел даёт научную нотацию.
// А Python print(time.time()) выводит как 1781912658.946944.
func formatValue(v starlark.Value) string {
	switch val := v.(type) {
	case starlark.Float:
		return strconv.FormatFloat(float64(val), 'f', 6, 64)
	case starlark.String:
		return string(val)
	default:
		return v.String()
	}
}

// ========================================
// Builtin-функции Starlark
// ========================================

// structBuiltin — аналог struct(...) в Starlark.
func structBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	dict := make(starlark.StringDict, len(kwargs))
	for _, kv := range kwargs {
		key := string(kv[0].(starlark.String))
		dict[key] = kv[1]
	}
	return starlarkstruct.FromStringDict(starlarkstruct.Default, dict), nil
}

// reFindall — аналог re.findall(pattern, text) в Starlark.
func reFindall(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pattern, text string
	if err := starlark.UnpackArgs("star_re_findall", args, kwargs,
		"pattern", &pattern,
		"text", &text,
	); err != nil {
		return nil, err
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("star_re_findall: %v", err)
	}
	matches := re.FindAllString(text, -1)
	elem := make([]starlark.Value, len(matches))
	for i, m := range matches {
		elem[i] = starlark.String(m)
	}
	return starlark.NewList(elem), nil
}

// reSearch — аналог re.search(pattern, text, group=0) в Starlark.
// Использует глобальный кеш скомпилированных regexp (аналог REGEX_CACHE в Python).
func reSearch(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pattern, text string
	group := 0
	if err := starlark.UnpackArgs("star_re_search", args, kwargs,
		"pattern", &pattern,
		"text", &text,
		"group?", &group,
	); err != nil {
		return nil, err
	}
	re := regexCache.get(pattern)
	match := re.FindStringSubmatch(text)
	if match == nil {
		return starlark.None, nil
	}
	if group >= len(match) {
		return starlark.None, nil
	}
	return starlark.String(match[group]), nil
}

// timeNow — аналог time.time() в Python.
func timeNow(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return starlark.Float(float64(time.Now().UnixMilli()) / 1000.0), nil
}

// timeSleep — аналог time.sleep(seconds) в Python.
// Принимает int или float.
func timeSleep(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var secs float64
	switch len(args) {
	case 0:
		return nil, fmt.Errorf("time_sleep: missing argument")
	default:
		switch v := args[0].(type) {
		case starlark.Float:
			secs = float64(v)
		case starlark.Int:
			n, ok := v.Int64()
			if !ok {
				return nil, fmt.Errorf("time_sleep: int overflow")
			}
			secs = float64(n)
		default:
			return nil, fmt.Errorf("time_sleep: expected float or int, got %s", v.Type())
		}
	}
	time.Sleep(time.Duration(secs * float64(time.Second)))
	return starlark.None, nil
}

// ========================================
// Утилиты
// ========================================

// regexCache — глобальный кеш скомпилированных regexp (аналог REGEX_CACHE в Python).
var regexCache = newRegexpCache()

type regexpCache struct {
	mu sync.RWMutex
	m  map[string]*regexp.Regexp
}

func newRegexpCache() *regexpCache {
	return &regexpCache{m: make(map[string]*regexp.Regexp)}
}

func (c *regexpCache) get(pattern string) *regexp.Regexp {
	c.mu.RLock()
	re, ok := c.m[pattern]
	c.mu.RUnlock()
	if ok {
		return re
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check
	if re, ok := c.m[pattern]; ok {
		return re
	}
	re = regexp.MustCompile(pattern)
	c.m[pattern] = re
	return re
}

// indent добавляет отступ к каждой строке текста.
// Аналог textwrap.indent() в Python.
func indent(text, prefix string) string {
	if text == "" {
		return prefix
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}
