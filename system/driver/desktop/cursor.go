// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"sync"

	"cogentcore.org/core/cursors/cursorimg"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// TheCursor is the single [system.Cursor] for the desktop platform
var TheCursor = &Cursor{CursorBase: system.CursorBase{Vis: true, Size: units.Dp(32)}, cursors: map[enums.Enum]map[int]*glfw.Cursor{}}

// Cursor is the [cursor.Cursor] implementation for the desktop platform
type Cursor struct {
	system.CursorBase

	// cursors are the cached glfw cursors
	cursors map[enums.Enum]map[int]*glfw.Cursor

	// mu is a mutex protecting access to the cursors
	mu sync.Mutex

	// prevSize is the cached previous size
	prevSize int

	// unitContext is the unit context used for converting the cursor size to dots.
	unitContext *units.Context
}

func (cu *Cursor) Set(cursor enums.Enum) error {
	cu.mu.Lock()
	defer cu.mu.Unlock()
	if cu.unitContext == nil {
		cu.unitContext = &units.Context{}
		cu.unitContext.Defaults()
	}
	cu.unitContext.DPI = TheApp.CtxWindow.LogDPI
	if TheApp.Platform() == system.MacOS {
		// The DPR is automatically applied to cursors on macOS already, so we cannot apply it here.
		cu.unitContext.DPI /= TheApp.CtxWindow.DevicePixelRatio
	}

	// If the cursorimg cache got cleared (probably due to a theme change),
	// we have to clear our cache as well.
	if len(cursorimg.Cursors) == 0 {
		clear(cu.cursors)
		cu.prevSize = 0
	}

	sizef := cu.Size.ToDots(cu.unitContext)
	// Multiples of 2 are less blurry on macOS Retina displays
	// (TODO: make cursors less blurry on macOS Retina displays:
	// https://github.com/cogentcore/core/issues/1373).
	size := int(math32.Round(sizef/2) * 2)
	if cursor == cu.Cur && size == cu.prevSize { // we already have, so we don't need to set again
		return nil
	}
	sm := cu.cursors[cursor]
	if sm == nil {
		sm = map[int]*glfw.Cursor{}
		cu.cursors[cursor] = sm
	}
	if cur, ok := sm[size]; ok {
		TheApp.CtxWindow.Glw.SetCursor(cur)
		cu.prevSize = size
		cu.Cur = cursor
		return nil
	}

	ci, err := cursorimg.Get(cursor, size)
	if err != nil {
		return err
	}
	h := ci.Hotspot
	gc := glfw.CreateCursor(ci.Image, h.X, h.Y)
	sm[size] = gc
	TheApp.CtxWindow.Glw.SetCursor(gc)
	cu.prevSize = size
	cu.Cur = cursor
	return nil
}
