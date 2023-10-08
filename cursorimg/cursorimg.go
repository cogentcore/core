// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cursorimg provides the cached rendering of SVG cursors to images.
package cursorimg

import (
	"fmt"
	"image"
	_ "image/png"

	"goki.dev/cursors"
	"goki.dev/enums"
)

// Cursor represents a cached rendered cursor, with the [image.Image]
// of the cursor and its hotspot.
type Cursor struct {
	// The cached image of the cursor.
	Image image.Image
	// The hotspot in terms of a percentage of the size
	// of the cursor from the top-left corner.
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
	// TODO: maybe support more sizes
	dir := ""
	switch size {
	case 32:
		dir = "32"
	case 64:
		dir = "64"
	default:
		return nil, fmt.Errorf("invalid cursor size %d; expected 32 or 64", size)
	}
	f, err := cursors.Cursors.Open("png/" + dir + "/" + name + ".png")
	if err != nil {
		return nil, fmt.Errorf("error opening PNG file for cursor %q: %w", name, err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("error reading PNG file for cursor %q: %w", name, err)
	}
	hot, ok := cursors.Hotspots[cursor]
	if !ok {
		// slog.Info("programmer error: missing cursor hotspot", "cursor", cursor)
		hot = image.Pt(100, 100)
	}
	return &Cursor{
		Image:   img,
		Hotspot: hot,
	}, nil

	// TODO: render from SVG at some point
	// sv := svg.NewSVG(size, size)
	// err := sv.OpenFS(cursors.Cursors, "svg/"+name+".svg") // TODO: support custom cursors
	// if err != nil {
	// 	err := fmt.Errorf("error opening SVG file for cursor %q: %w", name, err)
	// 	slog.Error(err.Error())
	// 	return nil, err
	// }
	// sv.SetNormXForm()
	// sv.Render()
	// return &Cursor{
	// 	Image: sv.Pixels,
	// }, nil
}
