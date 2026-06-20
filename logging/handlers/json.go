package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"runtime"
	"time"
)

// JSONHandler — кастомный slog.Handler для вывода в JSON.
// Аналог JsonFormatter из Python.
type JSONHandler struct {
	out       io.Writer
	addSource bool
}

// NewJSONHandler создаёт новый JSONHandler.
// По умолчанию addSource=true — будет добавлять "source" с file:line.
func NewJSONHandler(out io.Writer) *JSONHandler {
	return &JSONHandler{out: out, addSource: true}
}

func (h *JSONHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= slog.LevelDebug
}

func (h *JSONHandler) Handle(ctx context.Context, r slog.Record) error {
	m := make(map[string]any, r.NumAttrs()+5)
	m["time"] = r.Time.Format(time.RFC3339)
	m["level"] = r.Level.String()
	m["message"] = r.Message

	// Source info из PC (program counter)
	if h.addSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		m["source"] = map[string]any{
			"file": f.File,
			"line": f.Line,
		}
	}

	// Все дополнительные аттрибуты (ключ-значение)
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.Any()
		return true
	})

	return json.NewEncoder(h.out).Encode(m)
}

func (h *JSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // пока игнорируем — для простоты
}

func (h *JSONHandler) WithGroup(name string) slog.Handler {
	return h // пока игнорируем — для простоты
}
