package jsonpath

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Ошибки пакета.
var (
	ErrKeyNotFound  = errors.New("key not found")
	ErrTypeMismatch = errors.New("type mismatch: expected map or list")
	ErrInvalidPath  = errors.New("invalid path")
	ErrStopWalk     = errors.New("stop walk")
)

// Intent — намерение операции.
type Intent int

const (
	IntentGet Intent = iota
	IntentSet
	IntentDel
)

// walkConfig — опции обхода.
type walkConfig struct {
	defaultVal any
	silent     bool
}

// WalkOption — функциональная опция для Walk.
type WalkOption func(*walkConfig)

// WithDefault устанавливает значение по умолчанию для Get.
func WithDefault(val any) WalkOption {
	return func(c *walkConfig) {
		c.defaultVal = val
	}
}

// WithSilent управляет подавлением ошибок.
func WithSilent(silent bool) WalkOption {
	return func(c *walkConfig) {
		c.silent = silent
	}
}

// GetByPath — получить значение по пути.
// Возвращает список найденных значений.
func GetByPath(path string, data any, opts ...WalkOption) ([]any, error) {
	segments, err := ParsePath(path)
	if err != nil {
		return nil, err
	}
	cfg := &walkConfig{silent: true}
	for _, opt := range opts {
		opt(cfg)
	}
	return walk(data, segments, IntentGet, nil, cfg)
}

// SetByPath — установить значение по пути.
func SetByPath(path string, data any, value any, opts ...WalkOption) error {
	segments, err := ParsePath(path)
	if err != nil {
		return err
	}
	cfg := &walkConfig{silent: true}
	for _, opt := range opts {
		opt(cfg)
	}
	_, err = walk(data, segments, IntentSet, value, cfg)
	return err
}

// DelByPath — удалить значение по пути.
func DelByPath(path string, data any, opts ...WalkOption) error {
	segments, err := ParsePath(path)
	if err != nil {
		return err
	}
	cfg := &walkConfig{silent: true}
	for _, opt := range opts {
		opt(cfg)
	}
	_, err = walk(data, segments, IntentDel, nil, cfg)
	return err
}

// Rebuild — трансформация данных.
// Паттерны: "source.path->dest.path"
// Если -> отсутствует, dest = последний сегмент source.
func Rebuild(data any, patterns ...string) (any, error) {
	type pair struct {
		from string
		to   string
	}

	pairs := make([]pair, len(patterns))
	for i, p := range patterns {
		parts := strings.Split(p, "->")
		switch len(parts) {
		case 1:
			from := strings.TrimSpace(parts[0])
			segments, err := ParsePath(from)
			if err != nil {
				return nil, fmt.Errorf("rebuild pattern %q: %w", p, err)
			}
			last := segments[len(segments)-1]
			to := fmt.Sprintf("%v", last)
			pairs[i] = pair{from: from, to: to}
		case 2:
			pairs[i] = pair{
				from: strings.TrimSpace(parts[0]),
				to:   strings.TrimSpace(parts[1]),
			}
		default:
			return nil, fmt.Errorf("%w: invalid rebuild pattern %q", ErrInvalidPath, p)
		}
	}

	// Собираем значения из source-путей
	type collected struct {
		values []any
		toPath string
	}
	collectedList := make([]collected, len(pairs))

	for i, p := range pairs {
		vals, err := GetByPath(p.from, data)
		if err != nil {
			return nil, fmt.Errorf("rebuild get %q: %w", p.from, err)
		}
		collectedList[i] = collected{values: vals, toPath: p.to}
	}

	// Определяем структуру результата
	var result any
	for _, c := range collectedList {
		if len(c.values) > 0 {
			toSegments, err := ParsePath(c.toPath)
			if err != nil {
				return nil, fmt.Errorf("rebuild parse %q: %w", c.toPath, err)
			}
			if len(toSegments) > 1 {
				result = make([]any, len(c.values))
				for i := range result.([]any) {
					result.([]any)[i] = make(map[string]any)
				}
			} else {
				switch toSegments[0].(type) {
				case Index, Wildcard, Append:
					result = make([]any, 0)
				default:
					result = make(map[string]any)
				}
			}
			break
		}
	}

	if result == nil {
		return nil, nil
	}

	// Записываем значения в result
	for _, c := range collectedList {
		for idx, val := range c.values {
			toPath := c.toPath
			toPath = strings.Replace(toPath, "*", fmt.Sprintf("%d", idx), 1)
			if err := SetByPath(toPath, result, val); err != nil {
				return nil, fmt.Errorf("rebuild set %q: %w", toPath, err)
			}
		}
	}

	return result, nil
}

// DeepCopy — рекурсивное глубокое копирование any-значения.
// Поддерживает map[string]any, []any, string, int, float64, bool, nil.
func DeepCopy(src any) any {
	switch v := src.(type) {
	case map[string]any:
		dst := make(map[string]any, len(v))
		for k, val := range v {
			dst[k] = DeepCopy(val)
		}
		return dst
	case []any:
		dst := make([]any, len(v))
		for i, val := range v {
			dst[i] = DeepCopy(val)
		}
		return dst
	default:
		return v
	}
}

// DeepMerge — глубокое слияние двух map.
// copyOld=true (по умолчанию) — возвращает новый map, не меняя old.
func DeepMerge(old, new map[string]any, copyOld ...bool) map[string]any {
	copy := true
	if len(copyOld) > 0 {
		copy = copyOld[0]
	}

	var result map[string]any
	if copy {
		result = deepCopyMap(old)
	} else {
		result = old
	}

	deepMergeInto(result, new)
	return result
}

func deepCopyMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		if m, ok := v.(map[string]any); ok {
			dst[k] = deepCopyMap(m)
		} else {
			dst[k] = v
		}
	}
	return dst
}

func deepMergeInto(old, new map[string]any) {
	for k, v := range new {
		if newMap, ok := v.(map[string]any); ok {
			if oldMap, ok := old[k].(map[string]any); ok {
				deepMergeInto(oldMap, newMap)
			} else {
				old[k] = deepCopyMap(newMap)
			}
		} else {
			old[k] = v
		}
	}
}

// formatRegexp находит {key} но не {{key}} и не {key}}.
// В Go нет lookbehind/lookahead, поэтому используем подход:
// находим все {key}, потом фильтруем те, что не являются {{key}} или {key}}.
var formatRegexp = regexp.MustCompile(`\{([a-z\.0-9+\-_]+)\}`)

// Format — шаблонизация строки.
// {a.b.c} заменяется на значение по пути a.b.c из data.
// {{a.b.c}} и {a.b.c}} не заменяются (экранирование).
func Format(template string, data any) (string, error) {
	result := template
	matches := formatRegexp.FindAllStringSubmatchIndex(template, -1)

	// Идём с конца, чтобы не сбивать позиции
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		fullStart := match[0]
		fullEnd := match[1]
		keyStart := match[2]
		keyEnd := match[3]

		// Проверяем, не экранировано ли {{key}}
		if fullStart > 0 && template[fullStart-1] == '{' {
			continue
		}
		// Проверяем, не экранировано ли {key}}
		if fullEnd < len(template) && template[fullEnd] == '}' {
			continue
		}

		key := template[keyStart:keyEnd]
		vals, err := GetByPath(key, data)
		if err != nil {
			return "", fmt.Errorf("format: get %q: %w", key, err)
		}
		strs := make([]string, len(vals))
		for i, v := range vals {
			strs[i] = fmt.Sprintf("%v", v)
		}
		replacement := strings.Join(strs, ", ")
		result = result[:fullStart] + replacement + result[fullEnd:]
	}
	return result, nil
}

// DictToPaths — генерация путей из структуры данных.
// mode: "wild" (default), "strict", "full"
func DictToPaths(data any, mode ...string) ([]string, error) {
	m := "wild"
	if len(mode) > 0 {
		m = mode[0]
	}

	root := &pathNode{}
	if err := buildPaths(root, data, m); err != nil {
		return nil, err
	}
	return root.compile(), nil
}

// pathNode — узел дерева путей (внутренний тип для DictToPaths).
type pathNode struct {
	key      string
	children []*pathNode
}

func (n *pathNode) compile() []string {
	if len(n.children) == 0 {
		if n.key == "" {
			return nil
		}
		return []string{n.key}
	}

	var result []string
	seen := make(map[string]struct{})
	for _, child := range n.children {
		for _, sub := range child.compile() {
			full := sub
			if n.key != "" {
				full = n.key + "." + sub
			}
			if _, ok := seen[full]; !ok {
				seen[full] = struct{}{}
				result = append(result, full)
			}
		}
	}
	return result
}

func buildPaths(parent *pathNode, data any, mode string) error {
	switch v := data.(type) {
	case map[string]any:
		for k, val := range v {
			child := &pathNode{key: k}
			parent.children = append(parent.children, child)
			if err := buildPaths(child, val, mode); err != nil {
				return err
			}
		}
	case []any:
		if mode == "wild" || len(v) == 0 {
			child := &pathNode{key: "*"}
			parent.children = append(parent.children, child)
			if len(v) > 0 {
				if err := buildPaths(child, v[0], mode); err != nil {
					return err
				}
			}
		} else if mode == "strict" {
			for i, item := range v {
				child := &pathNode{key: fmt.Sprintf("%d", i)}
				parent.children = append(parent.children, child)
				if err := buildPaths(child, item, mode); err != nil {
					return err
				}
			}
		} else if mode == "full" {
			for _, item := range v {
				child := &pathNode{key: "*"}
				parent.children = append(parent.children, child)
				if err := buildPaths(child, item, mode); err != nil {
					return err
				}
			}
		}
	}
	return nil
}