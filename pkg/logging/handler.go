/*
Copyright 2025 The Kubernetes authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logging

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
)

const (
	ColorReset = "\033[0m"
	ColorError = "\033[31m"
	ColorWarn  = "\033[33m"
	ColorInfo  = "\033[36m"
	ColorDebug = "\033[32m"
)

type HandlerOptions struct {
	SlogOpts slog.HandlerOptions
}

type Handler struct {
	slog.Handler
	l *log.Logger
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Handler.Enabled(ctx, level)
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var color string
	switch r.Level {
	case slog.LevelDebug:
		color = ColorDebug + r.Level.String() + ColorReset
	case slog.LevelInfo:
		color = ColorInfo + r.Level.String() + ColorReset
	case slog.LevelWarn:
		color = ColorWarn + r.Level.String() + ColorReset
	case slog.LevelError:
		color = ColorError + r.Level.String() + ColorReset
	}
	attrs := ""
	r.Attrs(func(attr slog.Attr) bool {
		attrs += fmt.Sprintf("%s=%v", attr.Key, attr.Value.Any())
		return true
	})

	h.l.Println(color, r.Message, attrs)
	return nil
}

func (h *Handler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *Handler) WithGroup(_ string) slog.Handler {
	return h
}

func NewHandler(out io.Writer, opts HandlerOptions) *Handler {
	h := &Handler{
		Handler: slog.NewTextHandler(out, &opts.SlogOpts),
		l:       log.New(out, "", 0),
	}

	return h
}
