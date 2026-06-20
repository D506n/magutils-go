package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strings"
	"time"
)

// ColorHandler — кастомный slog.Handler для цветного вывода в консоль.
// Аналог ColoredConsoleFormatter из Python.
type ColorHandler struct {
	out         io.Writer
	levelColors map[slog.Level]string
	addSource   bool
}

var defaultLevelColors = map[slog.Level]string{
	slog.LevelDebug: "\033[36m", // Cyan
	slog.LevelInfo:  "\033[32m", // Green
	slog.LevelWarn:  "\033[33m", // Yellow
	slog.LevelError: "\033[31m", // Red
}

// NewColorHandler создаёт новый ColorHandler.
// По умолчанию addSource=true — будет показывать file:line.
func NewColorHandler(out io.Writer) *ColorHandler {
	return &ColorHandler{
		out:         out,
		levelColors: defaultLevelColors,
		addSource:   true,
	}
}

func (h *ColorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= slog.LevelDebug
}

func (h *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	reset := "\033[0m"
	lvlColor := h.levelColors[r.Level]

	// Получаем file:line из PC (program counter)
	file, line := "", 0
	if h.addSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		// Берём только имя файла, не полный путь
		if idx := strings.LastIndexByte(f.File, '/'); idx >= 0 {
			file = f.File[idx+1:]
		} else {
			file = f.File
		}
		line = f.Line
	}

	// Формат: [LEVEL|time|file:line] message
	if file != "" {
		fmt.Fprintf(h.out, "%s[%s]%s|%s|%s:%d] %s%s%s\n",
			lvlColor, r.Level.String(), reset,
			r.Time.Format(time.RFC3339),
			file, line,
			lvlColor, r.Message, reset,
		)
	} else {
		fmt.Fprintf(h.out, "%s[%s]%s|%s] %s%s%s\n",
			lvlColor, r.Level.String(), reset,
			r.Time.Format(time.RFC3339),
			lvlColor, r.Message, reset,
		)
	}
	return nil
}

func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // пока игнорируем — для простоты
}

func (h *ColorHandler) WithGroup(name string) slog.Handler {
	return h // пока игнорируем — для простоты
}
