package jsonpath

import (
	"fmt"
	"strconv"
	"strings"
)

// Segment — один сегмент разобранного пути.
type Segment interface {
	segmentMarker()
}

// Key — строковый ключ словаря.
type Key string

func (k Key) segmentMarker() {}

// Index — числовой индекс списка. Отрицательные значения — с конца.
type Index int

func (i Index) segmentMarker() {}

// Wildcard — "*" — все элементы списка.
type Wildcard struct{}

func (w Wildcard) segmentMarker() {}

// Append — "!a" — добавить в конец списка.
type Append struct{}

func (a Append) segmentMarker() {}

// ParsePath разбивает строку пути на сегменты.
// Пример: "a.b.0.*.c" -> [Key("a"), Key("b"), Index(0), Wildcard{}, Key("c")]
func ParsePath(path string) ([]Segment, error) {
	if path == "" {
		return nil, fmt.Errorf("%w: empty path", ErrInvalidPath)
	}

	parts := strings.Split(path, ".")
	segments := make([]Segment, 0, len(parts))

	for _, p := range parts {
		if p == "" {
			continue
		}

		switch p {
		case "*":
			segments = append(segments, Wildcard{})
		case "!a":
			segments = append(segments, Append{})
		default:
			// пробуем как число (в т.ч. отрицательное)
			if n, err := strconv.Atoi(p); err == nil {
				segments = append(segments, Index(n))
			} else {
				segments = append(segments, Key(p))
			}
		}
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("%w: no segments in path %q", ErrInvalidPath, path)
	}

	return segments, nil
}

// MustParsePath — как ParsePath, но паникует при ошибке.
func MustParsePath(path string) []Segment {
	segments, err := ParsePath(path)
	if err != nil {
		panic(err)
	}
	return segments
}