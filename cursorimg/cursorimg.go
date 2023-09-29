// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cursorimg provides the cached rendering of SVG cursors to images.
package cursorimg

import (
	"fmt"
	"image"
	"log/slog"

	"goki.dev/cursors"
	"goki.dev/svg"
)

// Cursor represents a cached rendered cursor, with the [image.Image]
// of the cursor and its hot spot.
type Cursor struct {
	Image   image.Image
	HotSpot image.Point
}

// Cursors contains all of the cached rendered cursors, specified first
// by name and then by size.
var Cursors = map[string]map[int]*Cursor{}

// Get returns the cursor object corresponding to the given cursor name,
// with the given size. If it is not already cached in [Cursors], it renders and caches it.
func Get(name string, size int) (*Cursor, error) {
	// TODO: move to prefs
	const cursorSize = 32
	sm := Cursors[name]
	if sm == nil {
		sm = map[int]*Cursor{}
	}
	if c, ok := sm[cursorSize]; ok {
		return c, nil
	}

	sv := svg.NewSVG(cursorSize, cursorSize)
	err := sv.OpenFS(cursors.Cursors, name+".svg") // TODO: support custom cursors
	if err != nil {
		err := fmt.Errorf("error opening SVG file for cursor %q: %w", name, err)
		slog.Error(err.Error())
		return nil, err
	}
	sv.SetNormXForm()
	sv.Render()
	return &Cursor{
		Image: sv.Pixels,
	}, nil
}
