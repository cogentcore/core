// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cursorimg provides the cached rendering of SVG cursors to images.
package cursorimg

import (
	"fmt"
	"image"
	_ "image/png"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/svg"
)

// Cursor represents a cached rendered cursor, with the [image.Image]
// of the cursor and its hotspot.
type Cursor struct {
	// The cached image of the cursor.
	Image image.Image
	// The size of the cursor.
	Size int
	// The hotspot is expressed in terms of raw cursor pixels.
	Hotspot image.Point
}

// Cursors contains all of the cached rendered cursors, specified first
// by cursor enum and then by size.
var Cursors = map[enums.Enum]map[int]*Cursor{}

// Get returns the cursor object corresponding to the given cursor enum,
// with the given size. If it is not already cached in [Cursors], it renders and caches it.
func Get(cursor enums.Enum, size int) (*Cursor, error) {
	sm := Cursors[cursor]
	if sm == nil {
		sm = map[int]*Cursor{}
		Cursors[cursor] = sm
	}
	if c, ok := sm[size]; ok {
		return c, nil
	}

	name := cursor.String()
	hot, ok := cursors.Hotspots[cursor]
	if !ok {
		hot = image.Pt(128, 128)
	}

	sv := svg.NewSVG(size, size)
	err := sv.OpenFS(cursors.Cursors, "svg/"+name+".svg")
	if err != nil {
		err := fmt.Errorf("error opening SVG file for cursor %q: %w", name, err)
		return nil, errors.Log(err)
	}
	sv.Render()
	return &Cursor{
		Image:   sv.Pixels,
		Size:    size,
		Hotspot: hot.Mul(size).Div(256),
	}, nil
}
