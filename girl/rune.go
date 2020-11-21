// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package girl

import (
	"errors"
	"image/color"

	"github.com/goki/gi/gist"
	"github.com/goki/mat32"
	"golang.org/x/image/font"
)

// Rune contains fully explicit data needed for rendering a single rune
// -- Face and Color can be nil after first element, in which case the last
// non-nil is used -- likely slightly more efficient to avoid setting all
// those pointers -- float32 values used to support better accuracy when
// transforming points
type Rune struct {
	Face    font.Face            `json:"-" xml:"-" desc:"fully-specified font rendering info, includes fully computed font size -- this is exactly what will be drawn -- no further transforms"`
	Color   color.Color          `json:"-" xml:"-" desc:"color to draw characters in"`
	BgColor color.Color          `json:"-" xml:"-" desc:"background color to fill background of color -- for highlighting, <mark> tag, etc -- unlike Face, Color, this must be non-nil for every case that uses it, as nil is also used for default transparent background"`
	Deco    gist.TextDecorations `desc:"additional decoration to apply -- underline, strike-through, etc -- also used for encoding a few special layout hints to pass info from styling tags to separate layout algorithms (e.g., &lt;P&gt; vs &lt;BR&gt;)"`
	RelPos  mat32.Vec2           `desc:"relative position from start of Text for the lower-left baseline rendering position of the font character"`
	Size    mat32.Vec2           `desc:"size of the rune itself, exclusive of spacing that might surround it"`
	RotRad  float32              `desc:"rotation in radians for this character, relative to its lower-left baseline rendering position"`
	ScaleX  float32              `desc:"scaling of the X dimension, in case of non-uniform scaling, 0 = no separate scaling"`
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
	// note: BgColor can be nil -- transparent
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
func (rr *Rune) CurColor(curColor color.Color) color.Color {
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
