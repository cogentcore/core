// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grog

import (
	"fmt"
	"log/slog"
)

// Print is equivalent to [fmt.Print], but with color based on the given level.
func Print(level slog.Level, a ...any) (n int, err error) {
	return fmt.Print(ApplyColor(level, fmt.Sprint(a...)))
}

// Println is equivalent to [fmt.Println], but with color based on the given level.
func Println(level slog.Level, a ...any) (n int, err error) {
	return fmt.Println(ApplyColor(level, fmt.Sprint(a...)))
}

// Printf is equivalent to [fmt.Printf], but with color based on the given level.
func Printf(level slog.Level, format string, a ...any) (n int, err error) {
	return fmt.Println(ApplyColor(level, fmt.Sprintf(format, a...)))
}
