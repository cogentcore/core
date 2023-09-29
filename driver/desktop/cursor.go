// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"goki.dev/enums"
	"goki.dev/goosi/cursor"
)

var theCursor = &cursorImpl{}

type cursorImpl struct {
	cursor.CursorBase
}

func (c *cursorImpl) Set(cursor enums.Enum) {

}
