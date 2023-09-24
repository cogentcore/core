// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This code is based on https://github.com/jba/slog/blob/main/handlers/loghandler/log_handler.go
// Copyright (c) 2022, Jonathan Amsterdam. All rights reserved. (BSD 3-Clause License)

package grog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"sync"

	"github.com/muesli/termenv"
	"goki.dev/matcolor"
)

// Handler is a [slog.Handler] whose output resembles that of [log.Logger].
// Use [NewHandler] to make a new [Handler] from a writer and options.
type Handler struct {
	Opts      slog.HandlerOptions
	Prefix    string // preformatted group names followed by a dot
	Preformat string // preformatted Attrs, with an initial space

	Mu sync.Mutex
	W  io.Writer
}

var _ slog.Handler = &Handler{}

// SetDefaultLogger sets the default logger to be a [Handler] with the
// level set to [UserLevel].
func SetDefaultLogger() {
	lvar := &slog.LevelVar{}
	lvar.Set(slog.Level(UserLevel))
	slog.SetDefault(slog.New(NewHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvar,
	})))
	if UseColor {
		restoreConsole, err := termenv.EnableVirtualTerminalProcessing(termenv.DefaultOutput())
		if err != nil {
			panic(err)
		}
		_ = restoreConsole
		if termenv.HasDarkBackground() {
			matcolor.TheScheme = &matcolor.TheSchemes.Dark
		} else {
			matcolor.TheScheme = &matcolor.TheSchemes.Light
		}
	}
}

// NewHandler makes a new [Handler] for the given writer with the given options.
func NewHandler(w io.Writer, opts *slog.HandlerOptions) *Handler {
	h := &Handler{W: w}
	if opts != nil {
		h.Opts = *opts
	}
	if h.Opts.ReplaceAttr == nil {
		h.Opts.ReplaceAttr = func(_ []string, a slog.Attr) slog.Attr { return a }
	}
	return h
}

// Enabled returns whether the handler should log a mesage with the given
// level in the given context.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.Opts.Level != nil {
		minLevel = h.Opts.Level.Level()
	}
	return level >= minLevel
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		W:         h.W,
		Opts:      h.Opts,
		Preformat: h.Preformat,
		Prefix:    h.Prefix + name + ".",
	}
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var buf []byte
	for _, a := range attrs {
		buf = h.AppendAttr(buf, h.Prefix, a)
	}
	return &Handler{
		W:         h.W,
		Opts:      h.Opts,
		Prefix:    h.Prefix,
		Preformat: h.Preformat + string(buf),
	}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	var buf []byte
	if !r.Time.IsZero() {
		buf = r.Time.AppendFormat(buf, "2006/01/02 15:04:05")
		buf = append(buf, ' ')
	}
	buf = append(buf, r.Level.String()...)
	buf = append(buf, ' ')
	if h.Opts.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		buf = append(buf, f.File...)
		buf = append(buf, ':')
		buf = strconv.AppendInt(buf, int64(f.Line), 10)
		buf = append(buf, ' ')
	}
	buf = append(buf, r.Message...)
	buf = append(buf, h.Preformat...)
	r.Attrs(func(a slog.Attr) bool {
		buf = h.AppendAttr(buf, h.Prefix, a)
		return true
	})
	buf = append(buf, '\n')
	h.Mu.Lock()
	defer h.Mu.Unlock()
	if UseColor {
		_, err := h.W.Write([]byte(termenv.String(string(buf)).Foreground(colorProfile.FromColor(matcolor.TheScheme.Primary)).String()))
		return err
	}
	_, err := h.W.Write(buf)
	return err
}

func (h *Handler) AppendAttr(buf []byte, prefix string, a slog.Attr) []byte {
	if a.Equal(slog.Attr{}) {
		return buf
	}
	if a.Value.Kind() != slog.KindGroup {
		buf = append(buf, ' ')
		buf = append(buf, prefix...)
		buf = append(buf, a.Key...)
		buf = append(buf, '=')
		return fmt.Appendf(buf, "%v", a.Value.Any())
	}
	// Group
	if a.Key != "" {
		prefix += a.Key + "."
	}
	for _, a := range a.Value.Group() {
		buf = h.AppendAttr(buf, prefix, a)
	}
	return buf
}
