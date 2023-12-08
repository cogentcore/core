// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"strings"
	"syscall/js"

	"goki.dev/cursors"
	"goki.dev/enums"
	"goki.dev/goosi/cursor"
)

// TheCursor is the single [goosi.Cursor] for the web platform
var TheCursor = Cursor{CursorBase: cursor.CursorBase{Vis: true, Size: 32}}

// Cursor is the [cursor.Cursor] implementation for the web platform
type Cursor struct {
	cursor.CursorBase
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
