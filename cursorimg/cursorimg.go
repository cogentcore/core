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
)

// Cursor represents a cached rendered cursor, with the [image.Image]
// of the cursor and its hot spot.
type Cursor struct {
	Image   image.Image
	Hotspot image.Point
}

// Cursors contains all of the cached rendered cursors, specified first
// by name and then by size.
var Cursors = map[string]map[int]*Cursor{}

// Get returns the cursor object corresponding to the given cursor name,
// with the given size. If it is not already cached in [Cursors], it renders and caches it.
func Get(name string, size int) (*Cursor, error) {
	sm := Cursors[name]
	if sm == nil {
		sm = map[int]*Cursor{}
		Cursors[name] = sm
	}
	if c, ok := sm[size]; ok {
		return c, nil
	}

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
	return &Cursor{
		Image: img,
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
