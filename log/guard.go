package log

import (
	"fmt"
	"regexp"

	"github.com/go-logr/logr"
)

type guard struct {
	next          logr.LogSink
	allowed       map[string]bool // nil means every key is allowed
	omitErrorText bool
	message       *regexp.Regexp // nil means every message is allowed
}

func newGuard(next logr.LogSink, cfg LogrConfig) (*guard, error) {
	g := &guard{next: next, omitErrorText: cfg.OmitErrorText}
	if len(cfg.AllowedKeys) > 0 {
		g.allowed = map[string]bool{}
		for _, key := range cfg.AllowedKeys {
			g.allowed[key] = true
		}
	}
	if cfg.MessagePattern != "" {
		pattern, err := regexp.Compile(cfg.MessagePattern)
		if err != nil {
			return nil, fmt.Errorf("log: message pattern: %w", err)
		}
		g.message = pattern
	}
	return g, nil
}

func (g *guard) with(next logr.LogSink) *guard {
	return &guard{next: next, allowed: g.allowed, omitErrorText: g.omitErrorText, message: g.message}
}

func (g *guard) Init(info logr.RuntimeInfo) { g.next.Init(info) }

func (g *guard) Enabled(level int) bool { return g.next.Enabled(level) }

func (g *guard) WithName(name string) logr.LogSink { return g.with(g.next.WithName(name)) }

func (g *guard) WithValues(keysAndValues ...any) logr.LogSink {
	return g.with(g.next.WithValues(g.filter(keysAndValues)...))
}

func (g *guard) Info(level int, msg string, keysAndValues ...any) {
	g.next.Info(level, g.checkMessage(msg), g.filter(keysAndValues)...)
}

func (g *guard) Error(err error, msg string, keysAndValues ...any) {
	if g.omitErrorText {
		err = nil
	}
	g.next.Error(err, g.checkMessage(msg), g.filter(keysAndValues)...)
}

func (g *guard) checkMessage(msg string) string {
	if g.message == nil || g.message.MatchString(msg) {
		return msg
	}
	return "invalid_message"
}

func (g *guard) filter(keysAndValues []any) []any {
	if g.allowed == nil {
		return keysAndValues
	}
	out := []any{}
	dropped := []string{}
	for i := 0; i < len(keysAndValues); i += 2 {
		key, isString := keysAndValues[i].(string)
		var value any
		if i+1 < len(keysAndValues) {
			value = keysAndValues[i+1]
		}
		if isString && g.allowed[key] && isPrimitive(value) {
			out = append(out, key, value)
			continue
		}
		if isString {
			dropped = append(dropped, key) // names only, never values
		}
	}
	if len(dropped) > 0 {
		out = append(out, "droppedKeys", dropped)
	}
	return out
}

func isPrimitive(v any) bool {
	switch v.(type) {
	case nil, string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	}
	return false
}
