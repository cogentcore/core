// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grog

import (
	"image/color"
	"log/slog"

	"github.com/muesli/termenv"
	"goki.dev/colors"
)

// UseColor is whether to use color in log messages.
// It is on by default.
var UseColor = true

// colorProfile is the termenv color profile, stored globally for convenience.
// It is set by [SetDefaultLogger] to [termenv.ColorProfile] if [UseColor] is true.
var colorProfile termenv.Profile

// ApplyColor applies the given color to the given string
// and returns the resulting string. If [UseColor] is set
// to false, it just returns the string it was passed.
func ApplyColor(clr color.Color, str string) string {
	if !UseColor {
		return str
	}
	return termenv.String(str).Foreground(colorProfile.FromColor(clr)).String()
}

// LevelColor applies the color associated with the given level to the
// given string and returns the resulting string. If [UseColor] is set
// to false, it just returns the string it was passed.
func LevelColor(level slog.Level, str string) string {
	var clr color.RGBA
	switch level {
	case slog.LevelDebug:
		return DebugColor(str)
	case slog.LevelInfo:
		return InfoColor(str)
	case slog.LevelWarn:
		return WarnColor(str)
	case slog.LevelError:
		return ErrorColor(str)
	}
	return ApplyColor(clr, str)
}

// DebugColor applies the color associated with the debug level to
// the given string and returns the resulting string. If [UseColor] is set
// to false, it just returns the string it was passed.
func DebugColor(str string) string {
	return ApplyColor(colors.Scheme.Tertiary.Base, str)
}

// InfoColor applies the color associated with the info level to
// the given string and returns the resulting string. Because the
// color associated with the info level is just white/black, it just
// returns the given string, but it exists for API consistency.
func InfoColor(str string) string {
	return str
}

// WarnColor applies the color associated with the warn level to
// the given string and returns the resulting string. If [UseColor] is set
// to false, it just returns the string it was passed.
func WarnColor(str string) string {
	return ApplyColor(colors.Scheme.Warn.Base, str)
}

// ErrorColor applies the color associated with the error level to
// the given string and returns the resulting string. If [UseColor] is set
// to false, it just returns the string it was passed.
func ErrorColor(str string) string {
	return ApplyColor(colors.Scheme.Error.Base, str)
}
