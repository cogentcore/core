// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"errors"
	"image"

	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
	"golang.org/x/image/font"
)

// Rune contains fully explicit data needed for rendering a single rune
// -- Face and Color can be nil after first element, in which case the last
// non-nil is used -- likely slightly more efficient to avoid setting all
// those pointers -- float32 values used to support better accuracy when
// transforming points
type Rune struct {

	// fully-specified font rendering info, includes fully computed font size.
	// This is exactly what will be drawn, with no further transforms.
	Face font.Face `json:"-"`

	// Color is the color to draw characters in
	Color image.Image `json:"-"`

	// background color to fill background of color, for highlighting,
	// <mark> tag, etc.  Unlike Face, Color, this must be non-nil for every case
	// that uses it, as nil is also used for default transparent background.
	Background image.Image `json:"-"`

	// dditional decoration to apply: underline, strike-through, etc.
	// Also used for encoding a few special layout hints to pass info
	// from styling tags to separate layout algorithms (e.g., &lt;P&gt; vs &lt;BR&gt;)
	Deco styles.TextDecorations

	// relative position from start of Text for the lower-left baseline
	// rendering position of the font character
	RelPos mat32.Vec2

	// size of the rune itself, exclusive of spacing that might surround it
	Size mat32.Vec2

	// rotation in radians for this character, relative to its lower-left
	// baseline rendering position
	RotRad float32

	// scaling of the X dimension, in case of non-uniform scaling, 0 = no separate scaling
	ScaleX float32
}

// HasNil returns error if any of the key info (face, color) is nil -- only
// the first element must be non-nil
func (rr *Rune) HasNil() error {
	if rr.Face == nil {
		return errors.New("gi.Rune: Face is nil")
	}
	if rr.Color == nil {
		return errors.New("gi.Rune: Color is nil")
	}
	// note: BackgroundColor can be nil -- transparent
	return nil
}

// CurFace is convenience for updating current font face if non-nil
func (rr *Rune) CurFace(curFace font.Face) font.Face {
	if rr.Face != nil {
		return rr.Face
	}
	return curFace
}

// CurColor is convenience for updating current color if non-nil
func (rr *Rune) CurColor(curColor image.Image) image.Image {
	if rr.Color != nil {
		return rr.Color
	}
	return curColor
}

// RelPosAfterLR returns the relative position after given rune for LR order: RelPos.X + Size.X
func (rr *Rune) RelPosAfterLR() float32 {
	return rr.RelPos.X + rr.Size.X
}

// RelPosAfterRL returns the relative position after given rune for RL order: RelPos.X - Size.X
func (rr *Rune) RelPosAfterRL() float32 {
	return rr.RelPos.X - rr.Size.X
}

// RelPosAfterTB returns the relative position after given rune for TB order: RelPos.Y + Size.Y
func (rr *Rune) RelPosAfterTB() float32 {
	return rr.RelPos.Y + rr.Size.Y
}
