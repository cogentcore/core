// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logx

import (
	"bytes"
	"context"
	"image/color"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// captureStdout returns what the given function prints to stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	old := os.Stdout
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	b, _ := io.ReadAll(r)
	return string(b)
}

// noColor turns off color, and sets the user level, for the test.
func noColor(t *testing.T, level slog.Level) {
	uc, ul := UseColor, UserLevel
	UseColor, UserLevel = false, level
	t.Cleanup(func() { UseColor, UserLevel = uc, ul })
}

func TestPrints(t *testing.T) {
	noColor(t, slog.LevelInfo)
	out := captureStdout(t, func() {
		PrintDebug("no ")
		PrintInfo("a ")
		PrintWarn("b ")
		PrintError("c")
		PrintlnDebug("no")
		PrintlnInfo("d")
		PrintlnWarn("e")
		PrintlnError("f")
		PrintfDebug("no %d", 1)
		PrintfInfo("g%d ", 1)
		PrintfWarn("h%d ", 2)
		PrintfError("i%d\n", 3)
	})
	assert.Equal(t, "a b cd\ne\nf\ng1 h2 i3\n", out, "debug is below the user level, and Printf adds no newline")
}

func TestHandler(t *testing.T) {
	noColor(t, slog.LevelInfo)
	var b bytes.Buffer
	h := NewHandler(&b, nil)
	assert.True(t, h.Enabled(context.Background(), slog.LevelInfo))
	assert.False(t, h.Enabled(context.Background(), slog.LevelDebug), "info by default")
	lg := slog.New(h.WithGroup("g").WithAttrs([]slog.Attr{slog.Int("n", 1)}))
	lg.Info("hello", "k", "v", slog.Group("sub", slog.Bool("ok", true)), slog.Attr{})
	assert.True(t, strings.HasSuffix(b.String(), " INFO hello g.n=1 g.k=v g.sub.ok=true\n"), b.String())

	b.Reset()
	lvl := slog.LevelDebug
	sh := NewHandler(&b, &slog.HandlerOptions{AddSource: true, Level: lvl})
	r := slog.NewRecord(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), slog.LevelDebug, "msg", pcHere())
	assert.NoError(t, sh.Handle(context.Background(), r))
	out := b.String()
	assert.True(t, strings.HasPrefix(out, "2026/01/02 03:04:05 DEBUG "), out)
	assert.Contains(t, out, "more_test.go:", "the source")
	assert.True(t, strings.HasSuffix(out, " msg\n"), out)
}

func TestColors(t *testing.T) {
	noColor(t, slog.LevelInfo)
	for _, f := range []func(string) string{DebugColor, InfoColor, WarnColor, ErrorColor, SuccessColor, CmdColor, TitleColor} {
		assert.Equal(t, "s", f("s"), "no color")
	}
	assert.Equal(t, "s", LevelColor(slog.Level(2), "s"))
	UseColor = true
	assert.Equal(t, "s", InfoColor("s"), "info is never colored")
	assert.Contains(t, ApplyColor(colorRed, "s"), "s")
	assert.NotNil(t, ColorScheme())
}

var colorRed = color.RGBA{255, 0, 0, 255}

// pcHere returns the program counter of its caller.
func pcHere() uintptr {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	return pcs[0]
}
