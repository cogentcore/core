// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image/color"
	"log"

	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"
)

// this is an alternative manual styling strategy -- styling takes a lot of time
// and this should be a lot faster

func StyleInhInit(val interface{}) (inh, init bool) {
	if str, ok := val.(string); ok {
		switch str {
		case "inherit":
			return true, false
		case "initial":
			return false, true
		default:
			return false, false
		}
	}
	return false, false
}

// StyleSetError reports that cannot set property of given key with given value
func StyleSetError(key string, val interface{}) {
	log.Printf("gi.Style error: cannot set key: %s from value: %v\n", key, val)
}

type StyleFunc func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D)

// This is a compiled map of all style funcs for Style object -- compiled from
// individual ones at startup
var StyleAllStyleFuncs map[string]StyleFunc

func (s *Style) StyleFromProps(par *Style, props ki.Props, vp *Viewport2D) {
	if StyleAllStyleFuncs == nil {
		StyleAllStyleFuncs = make(map[string]StyleFunc, 100)
		for ky, fn := range StyleStyleFuncs {
			StyleAllStyleFuncs[ky] = fn
		}
		for ky, fn := range StyleLayoutFuncs {
			StyleAllStyleFuncs[ky] = fn
		}
		for ky, fn := range StyleFontFuncs {
			StyleAllStyleFuncs[ky] = fn
		}
		for ky, fn := range StyleTextFuncs {
			StyleAllStyleFuncs[ky] = fn
		}
		for ky, fn := range StyleBorderFuncs {
			StyleAllStyleFuncs[ky] = fn
		}
		for ky, fn := range StyleOutlineFuncs {
			StyleAllStyleFuncs[ky] = fn
		}
		for ky, fn := range StyleShadowFuncs {
			StyleAllStyleFuncs[ky] = fn
		}
		// fmt.Printf("all style len: %d\n", len(StyleAllStyleFuncs))
	}

	pr := prof.Start("StyleFromProps")
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		if sfunc, ok := StyleAllStyleFuncs[key]; ok {
			sfunc(s, key, val, par, vp)
		}
	}
	pr.End()
}

/////////////////////////////////////////////////////////////////////////////////
//  Style

// StyleStyleFuncs are functions for styling the Style object itself
var StyleStyleFuncs = map[string]StyleFunc{
	"display": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Display = par.Display
			} else if init {
				s.Display = true
			}
			return
		}
		if bv, ok := kit.ToBool(val); ok {
			s.Display = bv
		}
	},
	"visible": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Visible = par.Visible
			} else if init {
				s.Visible = false
			}
			return
		}
		if bv, ok := kit.ToBool(val); ok {
			s.Visible = bv
		}
	},
	"inactive": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Inactive = par.Inactive
			} else if init {
				s.Inactive = false
			}
			return
		}
		if bv, ok := kit.ToBool(val); ok {
			s.Inactive = bv
		}
	},
	"pointer-events": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.PointerEvents = par.PointerEvents
			} else if init {
				s.PointerEvents = true
			}
			return
		}
		if bv, ok := kit.ToBool(val); ok {
			s.PointerEvents = bv
		}
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Layout

// StyleLayoutFuncs are functions for styling the LayoutStyle object
var StyleLayoutFuncs = map[string]StyleFunc{
	"z-index": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.ZIndex = par.Layout.ZIndex
			} else if init {
				s.Layout.ZIndex = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			s.Layout.ZIndex = int(iv)
		}
	},
	"horizontal-align": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.AlignH = par.Layout.AlignH
			} else if init {
				s.Layout.AlignH = AlignLeft
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Layout.AlignH.FromString(vt)
		case Align:
			s.Layout.AlignH = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Layout.AlignH = Align(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"vertical-align": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.AlignV = par.Layout.AlignV
			} else if init {
				s.Layout.AlignV = AlignMiddle
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Layout.AlignV.FromString(vt)
		case Align:
			s.Layout.AlignV = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Layout.AlignV = Align(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"x": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.PosX = par.Layout.PosX
			} else if init {
				s.Layout.PosX.Val = 0
			}
			return
		}
		s.Layout.PosX.SetIFace(val, key)
	},
	"y": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.PosY = par.Layout.PosY
			} else if init {
				s.Layout.PosY.Val = 0
			}
			return
		}
		s.Layout.PosY.SetIFace(val, key)
	},
	"width": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.Width = par.Layout.Width
			} else if init {
				s.Layout.Width.Val = 0
			}
			return
		}
		s.Layout.Width.SetIFace(val, key)
	},
	"height": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.Height = par.Layout.Height
			} else if init {
				s.Layout.Height.Val = 0
			}
			return
		}
		s.Layout.Height.SetIFace(val, key)
	},
	"max-width": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.MaxWidth = par.Layout.MaxWidth
			} else if init {
				s.Layout.MaxWidth.Val = 0
			}
			return
		}
		s.Layout.MaxWidth.SetIFace(val, key)
	},
	"max-height": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.MaxHeight = par.Layout.MaxHeight
			} else if init {
				s.Layout.MaxHeight.Val = 0
			}
			return
		}
		s.Layout.MaxHeight.SetIFace(val, key)
	},
	"min-width": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.MinWidth = par.Layout.MinWidth
			} else if init {
				s.Layout.MinWidth.Set(2, units.Px)
			}
			return
		}
		s.Layout.MinWidth.SetIFace(val, key)
	},
	"min-height": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.MinHeight = par.Layout.MinHeight
			} else if init {
				s.Layout.MinHeight.Set(2, units.Px)
			}
			return
		}
		s.Layout.MinHeight.SetIFace(val, key)
	},
	"margin": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.Margin = par.Layout.Margin
			} else if init {
				s.Layout.Margin.Val = 0
			}
			return
		}
		s.Layout.Margin.SetIFace(val, key)
	},
	"padding": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.Padding = par.Layout.Padding
			} else if init {
				s.Layout.Padding.Val = 0
			}
			return
		}
		s.Layout.Padding.SetIFace(val, key)
	},
	"overflow": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.Overflow = par.Layout.Overflow
			} else if init {
				s.Layout.Overflow = OverflowAuto
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Layout.Overflow.FromString(vt)
		case Overflow:
			s.Layout.Overflow = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Layout.Overflow = Overflow(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"columns": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Layout.Columns = par.Layout.Columns
			} else if init {
				s.Layout.Columns = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			s.Layout.Columns = int(iv)
		}
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Font

// StyleFontFuncs are functions for styling the FontStyle object
var StyleFontFuncs = map[string]StyleFunc{
	"color": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Font.Color = par.Font.Color
			} else if init {
				s.Font.Color.SetColor(color.Black)
			}
			return
		}
		s.Font.Color.SetIFace(val, vp, key)
	},
	"background-color": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Font.BgColor = par.Font.BgColor
			} else if init {
				s.Font.BgColor = ColorSpec{}
			}
			return
		}
		s.Font.BgColor.SetIFace(val, vp, key)
	},
	"opacity": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Font.Opacity = par.Font.Opacity
			} else if init {
				s.Font.Opacity = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			s.Font.Opacity = iv
		}
	},
	"font-size": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Font.Size = par.Font.Size
			} else if init {
				s.Font.Size.Set(12, units.Pt)
			}
			return
		}
		s.Font.Size.SetIFace(val, key)
	},
	"font-family": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Font.Family = par.Font.Family
			} else if init {
				s.Font.Family = "" // font has defaults
			}
			return
		}
		s.Font.Family = kit.ToString(val)
	},
	"font-style": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Font.Style = par.Font.Style
			} else if init {
				s.Font.Style = FontNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Font.Style.FromString(vt)
		case FontStyles:
			s.Font.Style = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Font.Style = FontStyles(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"font-weight": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Font.Weight = par.Font.Weight
			} else if init {
				s.Font.Weight = WeightNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Font.Weight.FromString(vt)
		case FontWeights:
			s.Font.Weight = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Font.Weight = FontWeights(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"font-stretch": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Font.Stretch = par.Font.Stretch
			} else if init {
				s.Font.Stretch = FontStrNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Font.Stretch.FromString(vt)
		case FontStretch:
			s.Font.Stretch = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Font.Stretch = FontStretch(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"font-variant": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Font.Variant = par.Font.Variant
			} else if init {
				s.Font.Variant = FontVarNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Font.Variant.FromString(vt)
		case FontVariants:
			s.Font.Variant = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Font.Variant = FontVariants(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"text-decoration": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Font.Deco = par.Font.Deco
			} else if init {
				s.Font.Deco = DecoNone
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Font.Deco.FromString(vt)
		case TextDecorations:
			s.Font.Deco = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Font.Deco = TextDecorations(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"baseline-shift": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Font.Shift = par.Font.Shift
			} else if init {
				s.Font.Shift = ShiftBaseline
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Font.Shift.FromString(vt)
		case BaselineShifts:
			s.Font.Shift = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Font.Shift = BaselineShifts(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Text

// StyleTextFuncs are functions for styling the TextStyle object
var StyleTextFuncs = map[string]StyleFunc{
	"text-align": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.Align = par.Text.Align
			} else if init {
				s.Text.Align = AlignLeft
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Text.Align.FromString(vt)
		case Align:
			s.Text.Align = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Text.Align = Align(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"text-anchor": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.Anchor = par.Text.Anchor
			} else if init {
				s.Text.Anchor = AnchorStart
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Text.Anchor.FromString(vt)
		case TextAnchors:
			s.Text.Anchor = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Text.Anchor = TextAnchors(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"letter-spacing": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.LetterSpacing = par.Text.LetterSpacing
			} else if init {
				s.Text.LetterSpacing.Val = 0
			}
			return
		}
		s.Text.LetterSpacing.SetIFace(val, key)
	},
	"word-spacing": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.WordSpacing = par.Text.WordSpacing
			} else if init {
				s.Text.WordSpacing.Val = 0
			}
			return
		}
		s.Text.WordSpacing.SetIFace(val, key)
	},
	"line-height": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.LineHeight = par.Text.LineHeight
			} else if init {
				s.Text.LineHeight = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			s.Text.LineHeight = iv
		}
	},
	"white-space": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.WhiteSpace = par.Text.WhiteSpace
			} else if init {
				s.Text.WhiteSpace = WhiteSpaceNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Text.WhiteSpace.FromString(vt)
		case WhiteSpaces:
			s.Text.WhiteSpace = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Text.WhiteSpace = WhiteSpaces(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"unicode-bidi": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.UnicodeBidi = par.Text.UnicodeBidi
			} else if init {
				s.Text.UnicodeBidi = BidiNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Text.UnicodeBidi.FromString(vt)
		case UnicodeBidi:
			s.Text.UnicodeBidi = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Text.UnicodeBidi = UnicodeBidi(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"direction": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.Direction = par.Text.Direction
			} else if init {
				s.Text.Direction = LRTB
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Text.Direction.FromString(vt)
		case TextDirections:
			s.Text.Direction = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Text.Direction = TextDirections(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"writing-mode": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.WritingMode = par.Text.WritingMode
			} else if init {
				s.Text.WritingMode = LRTB
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Text.WritingMode.FromString(vt)
		case TextDirections:
			s.Text.WritingMode = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Text.WritingMode = TextDirections(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"glyph-orientation-vertical": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.OrientationVert = par.Text.OrientationVert
			} else if init {
				s.Text.OrientationVert = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			s.Text.OrientationVert = iv
		}
	},
	"glyph-orientation-horizontal": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.OrientationHoriz = par.Text.OrientationHoriz
			} else if init {
				s.Text.OrientationHoriz = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			s.Text.OrientationHoriz = iv
		}
	},
	"text-indent": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.Indent = par.Text.Indent
			} else if init {
				s.Text.Indent.Val = 0
			}
			return
		}
		s.Text.Indent.SetIFace(val, key)
	},
	"para-spacing": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.ParaSpacing = par.Text.ParaSpacing
			} else if init {
				s.Text.ParaSpacing.Val = 0
			}
			return
		}
		s.Text.ParaSpacing.SetIFace(val, key)
	},
	"tab-size": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Text.TabSize = par.Text.TabSize
			} else if init {
				s.Text.TabSize = 4
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			s.Text.TabSize = int(iv)
		}
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Border

// StyleBorderFuncs are functions for styling the BorderStyle object
var StyleBorderFuncs = map[string]StyleFunc{
	"border-style": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Border.Style = par.Border.Style
			} else if init {
				s.Border.Style = BorderSolid
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Border.Style.FromString(vt)
		case BorderDrawStyle:
			s.Border.Style = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Border.Style = BorderDrawStyle(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"border-width": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Border.Width = par.Border.Width
			} else if init {
				s.Border.Width.Val = 0
			}
			return
		}
		s.Border.Width.SetIFace(val, key)
	},
	"border-radius": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Border.Radius = par.Border.Radius
			} else if init {
				s.Border.Radius.Val = 0
			}
			return
		}
		s.Border.Radius.SetIFace(val, key)
	},
	"border-color": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Border.Color = par.Font.Color
			} else if init {
				s.Border.Color.SetColor(color.Black)
			}
			return
		}
		s.Border.Color.SetIFace(val, vp, key)
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Outline

// StyleOutlineFuncs are functions for styling the OutlineStyle object
var StyleOutlineFuncs = map[string]StyleFunc{
	"outline-style": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Outline.Style = par.Outline.Style
			} else if init {
				s.Outline.Style = BorderNone
			}
			return
		}
		switch vt := val.(type) {
		case string:
			s.Outline.Style.FromString(vt)
		case BorderDrawStyle:
			s.Outline.Style = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Outline.Style = BorderDrawStyle(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"outline-width": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Outline.Width = par.Outline.Width
			} else if init {
				s.Outline.Width.Val = 0
			}
			return
		}
		s.Outline.Width.SetIFace(val, key)
	},
	"outline-radius": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Outline.Radius = par.Outline.Radius
			} else if init {
				s.Outline.Radius.Val = 0
			}
			return
		}
		s.Outline.Radius.SetIFace(val, key)
	},
	"outline-color": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.Outline.Color = par.Font.Color
			} else if init {
				s.Outline.Color.SetColor(color.Black)
			}
			return
		}
		s.Outline.Color.SetIFace(val, vp, key)
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Shadow

// StyleShadowFuncs are functions for styling the ShadowStyle object
var StyleShadowFuncs = map[string]StyleFunc{
	"box-shadow.h-offset": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.BoxShadow.HOffset = par.BoxShadow.HOffset
			} else if init {
				s.BoxShadow.HOffset.Val = 0
			}
			return
		}
		s.BoxShadow.HOffset.SetIFace(val, key)
	},
	"box-shadow.v-offset": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.BoxShadow.VOffset = par.BoxShadow.VOffset
			} else if init {
				s.BoxShadow.VOffset.Val = 0
			}
			return
		}
		s.BoxShadow.VOffset.SetIFace(val, key)
	},
	"box-shadow.blur": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.BoxShadow.Blur = par.BoxShadow.Blur
			} else if init {
				s.BoxShadow.Blur.Val = 0
			}
			return
		}
		s.BoxShadow.Blur.SetIFace(val, key)
	},
	"box-shadow.spread": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.BoxShadow.Spread = par.BoxShadow.Spread
			} else if init {
				s.BoxShadow.Spread.Val = 0
			}
			return
		}
		s.BoxShadow.Spread.SetIFace(val, key)
	},
	"box-shadow.color": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.BoxShadow.Color = par.Font.Color
			} else if init {
				s.BoxShadow.Color.SetColor(color.Black)
			}
			return
		}
		s.BoxShadow.Color.SetIFace(val, vp, key)
	},
	"box-shadow.inset": func(s *Style, key string, val interface{}, par *Style, vp *Viewport2D) {
		if inh, init := StyleInhInit(val); inh || init {
			if inh {
				s.BoxShadow.Inset = par.BoxShadow.Inset
			} else if init {
				s.BoxShadow.Inset = false
			}
			return
		}
		if bv, ok := kit.ToBool(val); ok {
			s.BoxShadow.Inset = bv
		}
	},
}
