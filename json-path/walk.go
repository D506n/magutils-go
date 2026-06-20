package jsonpath

import (
	"errors"
	"fmt"
)

// walk — единая функция обхода данных по пути.
func walk(data any, segments []Segment, intent Intent, value any, cfg *walkConfig) ([]any, error) {
	acc := &accumulator{cfg: cfg}
	_, err := walkRec(data, segments, 0, intent, value, acc)
	if err != nil {
		if errors.Is(err, ErrStopWalk) {
			if cfg.silent {
				return acc.result, nil
			}
			return nil, err
		}
		return nil, err
	}
	return acc.result, nil
}

// accumulator — аккумулятор результатов и конфиг.
type accumulator struct {
	cfg    *walkConfig
	result []any
}

// walkRec возвращает (возможно изменённое) значение на текущем уровне.
// Это позволяет корректно обрабатывать append и del на вложенных уровнях.
func walkRec(data any, segments []Segment, pos int, intent Intent, value any, acc *accumulator) (any, error) {
	if pos >= len(segments) {
		switch intent {
		case IntentGet:
			acc.result = append(acc.result, data)
			return data, nil
		case IntentSet:
			return value, nil
		case IntentDel:
			return nil, nil // сигнал удалить
		}
		return data, nil
	}

	segment := segments[pos]
	last := pos == len(segments)-1

	switch s := segment.(type) {
	case Wildcard:
		return walkWildcard(data, segments, pos, intent, value, acc)
	case Append:
		return walkAppend(data, segments, pos, intent, value, acc, last)
	case Key:
		return walkKey(data, string(s), segments, pos, intent, value, acc, last)
	case Index:
		return walkIndex(data, int(s), segments, pos, intent, value, acc, last)
	default:
		return data, fmt.Errorf("%w: unknown segment type %T", ErrInvalidPath, segment)
	}
}

// walkWildcard — обработка "*".
func walkWildcard(data any, segments []Segment, pos int, intent Intent, value any, acc *accumulator) (any, error) {
	list, ok := data.([]any)
	if !ok {
		return data, fmt.Errorf("%w: wildcard requires list, got %T", ErrTypeMismatch, data)
	}

	last := pos == len(segments)-1

	if last {
		switch intent {
		case IntentGet:
			acc.result = append(acc.result, list...)
			return data, nil
		case IntentSet:
			for i := range list {
				list[i] = value
			}
			return list, nil
		case IntentDel:
			return []any{}, nil
		}
	}

	// wildcard не на последней позиции
	for i, item := range list {
		newItem, err := walkRec(item, segments, pos+1, intent, value, acc)
		if err != nil {
			if errors.Is(err, ErrStopWalk) && acc.cfg.silent {
				continue
			}
			return data, err
		}
		list[i] = newItem
	}
	return list, nil
}

// walkAppend — обработка "!a".
func walkAppend(data any, segments []Segment, pos int, intent Intent, value any, acc *accumulator, last bool) (any, error) {
	list, ok := data.([]any)
	if !ok {
		return data, fmt.Errorf("%w: append requires list, got %T", ErrTypeMismatch, data)
	}

	if intent != IntentSet {
		return data, fmt.Errorf("%w: append only works with Set intent", ErrInvalidPath)
	}

	if last {
		return append(list, value), nil
	}

	// !a не на последней позиции — создаём вложенную структуру
	nextSeg := segments[pos+1]
	var newItem any
	switch nextSeg.(type) {
	case Key:
		newItem = make(map[string]any)
	case Index, Wildcard, Append:
		newItem = make([]any, 0)
	}

	child, err := walkRec(newItem, segments, pos+1, intent, value, acc)
	if err != nil {
		return data, err
	}
	return append(list, child), nil
}

// walkKey — обработка строкового ключа.
func walkKey(data any, key string, segments []Segment, pos int, intent Intent, value any, acc *accumulator, last bool) (any, error) {
	m, ok := data.(map[string]any)
	if !ok {
		return data, fmt.Errorf("%w: key %q requires map, got %T", ErrTypeMismatch, key, data)
	}

	val, exists := m[key]

	if !exists {
		if intent == IntentSet {
			if last {
				m[key] = value
				acc.result = append(acc.result, value)
				return m, nil
			}
			// Создаём промежуточную структуру
			nextSeg := segments[pos+1]
			switch nextSeg.(type) {
			case Key:
				m[key] = make(map[string]any)
			case Index, Wildcard, Append:
				m[key] = make([]any, 0)
			}
			val = m[key]
		} else if intent == IntentGet {
			if acc.cfg.defaultVal != nil {
				acc.result = append(acc.result, acc.cfg.defaultVal)
				return m, nil
			}
			if acc.cfg.silent {
				return m, ErrStopWalk
			}
			return m, fmt.Errorf("%w: key %q not found", ErrKeyNotFound, key)
		} else { // IntentDel
			if acc.cfg.silent {
				return m, ErrStopWalk
			}
			return m, fmt.Errorf("%w: key %q not found", ErrKeyNotFound, key)
		}
	}

	if last {
		switch intent {
		case IntentGet:
			acc.result = append(acc.result, val)
		case IntentSet:
			m[key] = value
			acc.result = append(acc.result, value)
		case IntentDel:
			delete(m, key)
		}
		return m, nil
	}

	newVal, err := walkRec(val, segments, pos+1, intent, value, acc)
	if err != nil {
		return m, err
	}
	m[key] = newVal
	return m, nil
}

// walkIndex — обработка числового индекса.
func walkIndex(data any, idx int, segments []Segment, pos int, intent Intent, value any, acc *accumulator, last bool) (any, error) {
	list, ok := data.([]any)
	if !ok {
		return data, fmt.Errorf("%w: index requires list, got %T", ErrTypeMismatch, data)
	}

	// Нормализуем отрицательный индекс
	origIdx := idx
	if idx < 0 {
		idx = len(list) + idx
	}

	if idx < 0 || idx >= len(list) {
		if intent == IntentSet {
			if last {
				return append(list, value), nil
			}
			// Создаём промежуточную структуру
			nextSeg := segments[pos+1]
			var newItem any
			switch nextSeg.(type) {
			case Key:
				newItem = make(map[string]any)
			case Index, Wildcard, Append:
				newItem = make([]any, 0)
			}
			child, err := walkRec(newItem, segments, pos+1, intent, value, acc)
			if err != nil {
				return data, err
			}
			return append(list, child), nil
		}
		if acc.cfg.silent {
			return data, ErrStopWalk
		}
		return data, fmt.Errorf("%w: index %d out of range (len=%d)", ErrKeyNotFound, origIdx, len(list))
	}

	if last {
		switch intent {
		case IntentGet:
			acc.result = append(acc.result, list[idx])
		case IntentSet:
			list[idx] = value
			acc.result = append(acc.result, value)
		case IntentDel:
			return append(list[:idx], list[idx+1:]...), nil
		}
		return list, nil
	}

	newVal, err := walkRec(list[idx], segments, pos+1, intent, value, acc)
	if err != nil {
		return data, err
	}
	list[idx] = newVal
	return list, nil
}