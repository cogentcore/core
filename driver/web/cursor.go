// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"syscall/js"

	"goki.dev/cursors"
	"goki.dev/enums"
	"goki.dev/goosi/cursor"
)

var theCursor = cursorImpl{CursorBase: cursor.CursorBase{Vis: true, Size: 32}}

type cursorImpl struct {
	cursor.CursorBase
}

func (c *cursorImpl) Set(cursor enums.Enum) error {
	s := cursor.String()
	// css calls it default, not arrow
	if cursor == cursors.Arrow {
		s = "default"
	}
	js.Global().Get("document").Get("body").Get("style").Set("cursor", s)
	return nil
}
