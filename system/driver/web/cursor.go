// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"strings"
	"syscall/js"

	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
)

// TheCursor is the single [system.Cursor] for the web platform
var TheCursor = &Cursor{CursorBase: system.CursorBase{Vis: true, Size: units.Dp(32)}}

// Cursor is the [system.Cursor] implementation for the web platform
type Cursor struct {
	system.CursorBase
}

func (cu *Cursor) Set(cursor enums.Enum) error {
	s := cursor.String()
	// css calls it default, not arrow
	if cursor == cursors.Arrow {
		s = "default"
	}
	// css puts resize at the end and we put it at the start
	if strings.HasPrefix(s, "resize-") {
		s = strings.TrimPrefix(s, "resize-")
		s += "-resize"
	}
	js.Global().Get("document").Get("body").Get("style").Set("cursor", s)
	return nil
}
