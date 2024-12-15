// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cursorimg provides the cached rendering of SVG cursors to images.
package cursorimg

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"io/fs"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/paint"
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
//
// It automatically replaces literal colors in svg with appropriate scheme colors as follows:
//   - #fff: [colors.Palette].Neutral.ToneUniform(100)
//   - #000: [colors.Palette].Neutral.ToneUniform(0)
//   - #f00: [colors.Scheme].Error.Base
//   - #0f0: [colors.Scheme].Success.Base
//   - #ff0: [colors.Scheme].Warn.Base
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
	b, err := fs.ReadFile(cursors.Cursors, "svg/"+name+".svg")
	if err != nil {
		return nil, err
	}
	b = replaceColors(b)
	err = sv.ReadXML(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("error opening SVG file for cursor %q: %w", name, err)
	}
	sv.Render()

	blurRadius := size / 16
	bounds := sv.Pixels.Bounds()
	// We need to add extra space so that the shadow doesn't get clipped.
	bounds.Max = bounds.Max.Add(image.Pt(blurRadius, blurRadius))
	shadow := image.NewRGBA(bounds)
	draw.DrawMask(shadow, shadow.Bounds(), gradient.ApplyOpacity(colors.Scheme.Shadow, 0.25), image.Point{}, sv.Pixels, image.Point{}, draw.Src)
	shadow = paint.GaussianBlur(shadow, float64(blurRadius))
	draw.Draw(shadow, shadow.Bounds(), sv.Pixels, image.Point{}, draw.Over)

	return &Cursor{
		Image:   shadow,
		Size:    size,
		Hotspot: hot.Mul(size).Div(256),
	}, nil
}

// replaceColors replaces literal cursor colors in the given SVG with scheme colors.
func replaceColors(b []byte) []byte {
	m := map[string]image.Image{
		"#fff": colors.Palette.Neutral.ToneUniform(100),
		"#000": colors.Palette.Neutral.ToneUniform(0),
		"#f00": colors.Scheme.Error.Base,
		"#0f0": colors.Scheme.Success.Base,
		"#ff0": colors.Scheme.Warn.Base,
	}
	for old, clr := range m {
		b = bytes.ReplaceAll(b, []byte(fmt.Sprintf("%q", old)), []byte(fmt.Sprintf("%q", colors.AsHex(colors.ToUniform(clr)))))
	}
	return b
}
