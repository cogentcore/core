// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image/color"
	"log"
	"reflect"
	"strings"
	"unsafe"

	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"
)

// this is an alternative manual styling strategy -- styling takes a lot of time
// and this should be a lot faster

func StyleInhInit(val, par interface{}) (inh, init bool) {
	if str, ok := val.(string); ok {
		switch str {
		case "inherit":
			return !kit.IfaceIsNil(par), false
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

type StyleFunc func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D)

// StyleFromProps sets style field values based on ki.Props properties
func (s *Style) StyleFromProps(par *Style, props ki.Props, vp *Viewport2D) {
	// pr := prof.Start("StyleFromProps")
	// defer pr.End()
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		if sfunc, ok := StyleLayoutFuncs[key]; ok {
			if par != nil {
				sfunc(&s.Layout, key, val, &par.Layout, vp)
			} else {
				sfunc(&s.Layout, key, val, nil, vp)
			}
			continue
		}
		if sfunc, ok := StyleFontFuncs[key]; ok {
			if par != nil {
				sfunc(&s.Font, key, val, &par.Font, vp)
			} else {
				sfunc(&s.Font, key, val, nil, vp)
			}
			continue
		}
		if sfunc, ok := StyleTextFuncs[key]; ok {
			if par != nil {
				sfunc(&s.Text, key, val, &par.Text, vp)
			} else {
				sfunc(&s.Text, key, val, nil, vp)
			}
			continue
		}
		if sfunc, ok := StyleBorderFuncs[key]; ok {
			if par != nil {
				sfunc(&s.Border, key, val, &par.Border, vp)
			} else {
				sfunc(&s.Border, key, val, nil, vp)
			}
			continue
		}
		if sfunc, ok := StyleStyleFuncs[key]; ok {
			sfunc(s, key, val, par, vp)
			continue
		}
		if sfunc, ok := StyleOutlineFuncs[key]; ok {
			if par != nil {
				sfunc(&s.Outline, key, val, &par.Outline, vp)
			} else {
				sfunc(&s.Outline, key, val, nil, vp)
			}
			continue
		}
		if sfunc, ok := StyleShadowFuncs[key]; ok {
			if par != nil {
				sfunc(&s.BoxShadow, key, val, &par.BoxShadow, vp)
			} else {
				sfunc(&s.BoxShadow, key, val, nil, vp)
			}
			continue
		}
	}
}

/////////////////////////////////////////////////////////////////////////////////
//  ToDots

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (s *Style) ToDots(uc *units.Context) {
	s.StyleToDots(uc)
	s.Layout.ToDots(uc)
	s.Font.ToDots(uc)
	s.Text.ToDots(uc)
	s.Border.ToDots(uc)
	s.Outline.ToDots(uc)
	s.BoxShadow.ToDots(uc)
}

/////////////////////////////////////////////////////////////////////////////////
//  Style

// StyleStyleFuncs are functions for styling the Style object itself
var StyleStyleFuncs = map[string]StyleFunc{
	"display": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.Display = par.(*Style).Display
			} else if init {
				s.Display = true
			}
			return
		}
		if bv, ok := kit.ToBool(val); ok {
			s.Display = bv
		}
	},
	"visible": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.Visible = par.(*Style).Visible
			} else if init {
				s.Visible = false
			}
			return
		}
		if bv, ok := kit.ToBool(val); ok {
			s.Visible = bv
		}
	},
	"inactive": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.Inactive = par.(*Style).Inactive
			} else if init {
				s.Inactive = false
			}
			return
		}
		if bv, ok := kit.ToBool(val); ok {
			s.Inactive = bv
		}
	},
	"pointer-events": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.PointerEvents = par.(*Style).PointerEvents
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

// StyleToDots runs ToDots on unit values, to compile down to raw pixels
func (s *Style) StyleToDots(uc *units.Context) {
	// none
}

/////////////////////////////////////////////////////////////////////////////////
//  Layout

// StyleLayoutFuncs are functions for styling the LayoutStyle object
var StyleLayoutFuncs = map[string]StyleFunc{
	"z-index": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.ZIndex = par.(*LayoutStyle).ZIndex
			} else if init {
				ly.ZIndex = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ly.ZIndex = int(iv)
		}
	},
	"horizontal-align": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.AlignH = par.(*LayoutStyle).AlignH
			} else if init {
				ly.AlignH = AlignLeft
			}
			return
		}
		switch vt := val.(type) {
		case string:
			ly.AlignH.FromString(vt)
		case Align:
			ly.AlignH = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				ly.AlignH = Align(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"vertical-align": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.AlignV = par.(*LayoutStyle).AlignV
			} else if init {
				ly.AlignV = AlignMiddle
			}
			return
		}
		switch vt := val.(type) {
		case string:
			ly.AlignV.FromString(vt)
		case Align:
			ly.AlignV = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				ly.AlignV = Align(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"x": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.PosX = par.(*LayoutStyle).PosX
			} else if init {
				ly.PosX.Val = 0
			}
			return
		}
		ly.PosX.SetIFace(val, key)
	},
	"y": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.PosY = par.(*LayoutStyle).PosY
			} else if init {
				ly.PosY.Val = 0
			}
			return
		}
		ly.PosY.SetIFace(val, key)
	},
	"width": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Width = par.(*LayoutStyle).Width
			} else if init {
				ly.Width.Val = 0
			}
			return
		}
		ly.Width.SetIFace(val, key)
	},
	"height": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Height = par.(*LayoutStyle).Height
			} else if init {
				ly.Height.Val = 0
			}
			return
		}
		ly.Height.SetIFace(val, key)
	},
	"max-width": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.MaxWidth = par.(*LayoutStyle).MaxWidth
			} else if init {
				ly.MaxWidth.Val = 0
			}
			return
		}
		ly.MaxWidth.SetIFace(val, key)
	},
	"max-height": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.MaxHeight = par.(*LayoutStyle).MaxHeight
			} else if init {
				ly.MaxHeight.Val = 0
			}
			return
		}
		ly.MaxHeight.SetIFace(val, key)
	},
	"min-width": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.MinWidth = par.(*LayoutStyle).MinWidth
			} else if init {
				ly.MinWidth.Set(2, units.Px)
			}
			return
		}
		ly.MinWidth.SetIFace(val, key)
	},
	"min-height": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.MinHeight = par.(*LayoutStyle).MinHeight
			} else if init {
				ly.MinHeight.Set(2, units.Px)
			}
			return
		}
		ly.MinHeight.SetIFace(val, key)
	},
	"margin": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Margin = par.(*LayoutStyle).Margin
			} else if init {
				ly.Margin.Val = 0
			}
			return
		}
		ly.Margin.SetIFace(val, key)
	},
	"padding": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Padding = par.(*LayoutStyle).Padding
			} else if init {
				ly.Padding.Val = 0
			}
			return
		}
		ly.Padding.SetIFace(val, key)
	},
	"overflow": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Overflow = par.(*LayoutStyle).Overflow
			} else if init {
				ly.Overflow = OverflowAuto
			}
			return
		}
		switch vt := val.(type) {
		case string:
			ly.Overflow.FromString(vt)
		case Overflow:
			ly.Overflow = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				ly.Overflow = Overflow(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"columns": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Columns = par.(*LayoutStyle).Columns
			} else if init {
				ly.Columns = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ly.Columns = int(iv)
		}
	},
	"row": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Row = par.(*LayoutStyle).Row
			} else if init {
				ly.Row = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ly.Row = int(iv)
		}
	},
	"col": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Col = par.(*LayoutStyle).Col
			} else if init {
				ly.Col = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ly.Col = int(iv)
		}
	},
	"row-span": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.RowSpan = par.(*LayoutStyle).RowSpan
			} else if init {
				ly.RowSpan = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ly.RowSpan = int(iv)
		}
	},
	"col-span": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.ColSpan = par.(*LayoutStyle).ColSpan
			} else if init {
				ly.ColSpan = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ly.ColSpan = int(iv)
		}
	},
	"scrollbar-width": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ly := obj.(*LayoutStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.ScrollBarWidth = par.(*LayoutStyle).ScrollBarWidth
			} else if init {
				ly.ScrollBarWidth.Val = 0
			}
			return
		}
		ly.ScrollBarWidth.SetIFace(val, key)
	},
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (ly *LayoutStyle) ToDots(uc *units.Context) {
	ly.PosX.ToDots(uc)
	ly.PosY.ToDots(uc)
	ly.Width.ToDots(uc)
	ly.Height.ToDots(uc)
	ly.MaxWidth.ToDots(uc)
	ly.MaxHeight.ToDots(uc)
	ly.MinWidth.ToDots(uc)
	ly.MinHeight.ToDots(uc)
	ly.Margin.ToDots(uc)
	ly.Padding.ToDots(uc)
	ly.ScrollBarWidth.ToDots(uc)
}

/////////////////////////////////////////////////////////////////////////////////
//  Font

// StyleFontFuncs are functions for styling the FontStyle object
var StyleFontFuncs = map[string]StyleFunc{
	"color": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FontStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*FontStyle).Color
			} else if init {
				fs.Color.SetColor(color.Black)
			}
			return
		}
		fs.Color.SetIFace(val, vp, key)
	},
	"background-color": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FontStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.BgColor = par.(*FontStyle).BgColor
			} else if init {
				fs.BgColor = ColorSpec{}
			}
			return
		}
		fs.BgColor.SetIFace(val, vp, key)
	},
	"opacity": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FontStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Opacity = par.(*FontStyle).Opacity
			} else if init {
				fs.Opacity = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			fs.Opacity = iv
		}
	},
	"font-size": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FontStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Size = par.(*FontStyle).Size
			} else if init {
				fs.Size.Set(12, units.Pt)
			}
			return
		}
		fs.Size.SetIFace(val, key)
	},
	"font-family": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FontStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Family = par.(*FontStyle).Family
			} else if init {
				fs.Family = "" // font has defaults
			}
			return
		}
		fs.Family = kit.ToString(val)
	},
	"font-style": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FontStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Style = par.(*FontStyle).Style
			} else if init {
				fs.Style = FontNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			fs.Style.FromString(vt)
		case FontStyles:
			fs.Style = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				fs.Style = FontStyles(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"font-weight": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FontStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Weight = par.(*FontStyle).Weight
			} else if init {
				fs.Weight = WeightNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			fs.Weight.FromString(vt)
		case FontWeights:
			fs.Weight = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				fs.Weight = FontWeights(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"font-stretch": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FontStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Stretch = par.(*FontStyle).Stretch
			} else if init {
				fs.Stretch = FontStrNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			fs.Stretch.FromString(vt)
		case FontStretch:
			fs.Stretch = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				fs.Stretch = FontStretch(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"font-variant": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FontStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Variant = par.(*FontStyle).Variant
			} else if init {
				fs.Variant = FontVarNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			fs.Variant.FromString(vt)
		case FontVariants:
			fs.Variant = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				fs.Variant = FontVariants(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"text-decoration": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FontStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Deco = par.(*FontStyle).Deco
			} else if init {
				fs.Deco = DecoNone
			}
			return
		}
		switch vt := val.(type) {
		case string:
			fs.Deco.FromString(vt)
		case TextDecorations:
			fs.Deco = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				fs.Deco = TextDecorations(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"baseline-shift": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FontStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Shift = par.(*FontStyle).Shift
			} else if init {
				fs.Shift = ShiftBaseline
			}
			return
		}
		switch vt := val.(type) {
		case string:
			fs.Shift.FromString(vt)
		case BaselineShifts:
			fs.Shift = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				fs.Shift = BaselineShifts(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (fs *FontStyle) ToDots(uc *units.Context) {
	fs.Size.ToDots(uc)
}

/////////////////////////////////////////////////////////////////////////////////
//  Text

// StyleTextFuncs are functions for styling the TextStyle object
var StyleTextFuncs = map[string]StyleFunc{
	"text-align": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.Align = par.(*TextStyle).Align
			} else if init {
				ts.Align = AlignLeft
			}
			return
		}
		switch vt := val.(type) {
		case string:
			ts.Align.FromString(vt)
		case Align:
			ts.Align = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				ts.Align = Align(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"text-anchor": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.Anchor = par.(*TextStyle).Anchor
			} else if init {
				ts.Anchor = AnchorStart
			}
			return
		}
		switch vt := val.(type) {
		case string:
			ts.Anchor.FromString(vt)
		case TextAnchors:
			ts.Anchor = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				ts.Anchor = TextAnchors(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"letter-spacing": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.LetterSpacing = par.(*TextStyle).LetterSpacing
			} else if init {
				ts.LetterSpacing.Val = 0
			}
			return
		}
		ts.LetterSpacing.SetIFace(val, key)
	},
	"word-spacing": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.WordSpacing = par.(*TextStyle).WordSpacing
			} else if init {
				ts.WordSpacing.Val = 0
			}
			return
		}
		ts.WordSpacing.SetIFace(val, key)
	},
	"line-height": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.LineHeight = par.(*TextStyle).LineHeight
			} else if init {
				ts.LineHeight = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			ts.LineHeight = iv
		}
	},
	"white-space": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.WhiteSpace = par.(*TextStyle).WhiteSpace
			} else if init {
				ts.WhiteSpace = WhiteSpaceNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			ts.WhiteSpace.FromString(vt)
		case WhiteSpaces:
			ts.WhiteSpace = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				ts.WhiteSpace = WhiteSpaces(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"unicode-bidi": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.UnicodeBidi = par.(*TextStyle).UnicodeBidi
			} else if init {
				ts.UnicodeBidi = BidiNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			ts.UnicodeBidi.FromString(vt)
		case UnicodeBidi:
			ts.UnicodeBidi = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				ts.UnicodeBidi = UnicodeBidi(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"direction": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.Direction = par.(*TextStyle).Direction
			} else if init {
				ts.Direction = LRTB
			}
			return
		}
		switch vt := val.(type) {
		case string:
			ts.Direction.FromString(vt)
		case TextDirections:
			ts.Direction = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				ts.Direction = TextDirections(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"writing-mode": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.WritingMode = par.(*TextStyle).WritingMode
			} else if init {
				ts.WritingMode = LRTB
			}
			return
		}
		switch vt := val.(type) {
		case string:
			ts.WritingMode.FromString(vt)
		case TextDirections:
			ts.WritingMode = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				ts.WritingMode = TextDirections(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"glyph-orientation-vertical": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.OrientationVert = par.(*TextStyle).OrientationVert
			} else if init {
				ts.OrientationVert = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			ts.OrientationVert = iv
		}
	},
	"glyph-orientation-horizontal": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.OrientationHoriz = par.(*TextStyle).OrientationHoriz
			} else if init {
				ts.OrientationHoriz = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			ts.OrientationHoriz = iv
		}
	},
	"text-indent": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.Indent = par.(*TextStyle).Indent
			} else if init {
				ts.Indent.Val = 0
			}
			return
		}
		ts.Indent.SetIFace(val, key)
	},
	"para-spacing": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.ParaSpacing = par.(*TextStyle).ParaSpacing
			} else if init {
				ts.ParaSpacing.Val = 0
			}
			return
		}
		ts.ParaSpacing.SetIFace(val, key)
	},
	"tab-size": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ts := obj.(*TextStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.TabSize = par.(*TextStyle).TabSize
			} else if init {
				ts.TabSize = 4
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ts.TabSize = int(iv)
		}
	},
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (ts *TextStyle) ToDots(uc *units.Context) {
	ts.LetterSpacing.ToDots(uc)
	ts.WordSpacing.ToDots(uc)
	ts.Indent.ToDots(uc)
	ts.ParaSpacing.ToDots(uc)
}

/////////////////////////////////////////////////////////////////////////////////
//  Border

// StyleBorderFuncs are functions for styling the BorderStyle object
var StyleBorderFuncs = map[string]StyleFunc{
	"border-style": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		bs := obj.(*BorderStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Style = par.(*BorderStyle).Style
			} else if init {
				bs.Style = BorderSolid
			}
			return
		}
		switch vt := val.(type) {
		case string:
			bs.Style.FromString(vt)
		case BorderDrawStyle:
			bs.Style = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				bs.Style = BorderDrawStyle(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"border-width": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		bs := obj.(*BorderStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Width = par.(*BorderStyle).Width
			} else if init {
				bs.Width.Val = 0
			}
			return
		}
		bs.Width.SetIFace(val, key)
	},
	"border-radius": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		bs := obj.(*BorderStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Radius = par.(*BorderStyle).Radius
			} else if init {
				bs.Radius.Val = 0
			}
			return
		}
		bs.Radius.SetIFace(val, key)
	},
	"border-color": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		bs := obj.(*BorderStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Color = par.(*BorderStyle).Color
			} else if init {
				bs.Color.SetColor(color.Black)
			}
			return
		}
		bs.Color.SetIFace(val, vp, key)
	},
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (bs *BorderStyle) ToDots(uc *units.Context) {
	bs.Width.ToDots(uc)
	bs.Radius.ToDots(uc)
}

/////////////////////////////////////////////////////////////////////////////////
//  Outline

// StyleOutlineFuncs are functions for styling the OutlineStyle object
var StyleOutlineFuncs = map[string]StyleFunc{
	"outline-style": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		bs := obj.(*BorderStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Style = par.(*BorderStyle).Style
			} else if init {
				bs.Style = BorderNone
			}
			return
		}
		switch vt := val.(type) {
		case string:
			bs.Style.FromString(vt)
		case BorderDrawStyle:
			bs.Style = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				bs.Style = BorderDrawStyle(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"outline-width": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		bs := obj.(*BorderStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Width = par.(*BorderStyle).Width
			} else if init {
				bs.Width.Val = 0
			}
			return
		}
		bs.Width.SetIFace(val, key)
	},
	"outline-radius": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		bs := obj.(*BorderStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Radius = par.(*BorderStyle).Radius
			} else if init {
				bs.Radius.Val = 0
			}
			return
		}
		bs.Radius.SetIFace(val, key)
	},
	"outline-color": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		bs := obj.(*BorderStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Color = par.(*BorderStyle).Color
			} else if init {
				bs.Color.SetColor(color.Black)
			}
			return
		}
		bs.Color.SetIFace(val, vp, key)
	},
}

// Note: uses BorderStyle.ToDots for now

/////////////////////////////////////////////////////////////////////////////////
//  Shadow

// StyleShadowFuncs are functions for styling the ShadowStyle object
var StyleShadowFuncs = map[string]StyleFunc{
	"box-shadow.h-offset": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ss := obj.(*ShadowStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.HOffset = par.(*ShadowStyle).HOffset
			} else if init {
				ss.HOffset.Val = 0
			}
			return
		}
		ss.HOffset.SetIFace(val, key)
	},
	"box-shadow.v-offset": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ss := obj.(*ShadowStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.VOffset = par.(*ShadowStyle).VOffset
			} else if init {
				ss.VOffset.Val = 0
			}
			return
		}
		ss.VOffset.SetIFace(val, key)
	},
	"box-shadow.blur": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ss := obj.(*ShadowStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.Blur = par.(*ShadowStyle).Blur
			} else if init {
				ss.Blur.Val = 0
			}
			return
		}
		ss.Blur.SetIFace(val, key)
	},
	"box-shadow.spread": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ss := obj.(*ShadowStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.Spread = par.(*ShadowStyle).Spread
			} else if init {
				ss.Spread.Val = 0
			}
			return
		}
		ss.Spread.SetIFace(val, key)
	},
	"box-shadow.color": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ss := obj.(*ShadowStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.Color = par.(*ShadowStyle).Color
			} else if init {
				ss.Color.SetColor(color.Black)
			}
			return
		}
		ss.Color.SetIFace(val, vp, key)
	},
	"box-shadow.inset": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		ss := obj.(*ShadowStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.Inset = par.(*ShadowStyle).Inset
			} else if init {
				ss.Inset = false
			}
			return
		}
		if bv, ok := kit.ToBool(val); ok {
			ss.Inset = bv
		}
	},
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (bs *ShadowStyle) ToDots(uc *units.Context) {
	bs.HOffset.ToDots(uc)
	bs.VOffset.ToDots(uc)
	bs.Blur.ToDots(uc)
	bs.Spread.ToDots(uc)
}

////////////////////////////////////////////////////////////////////////////////////////
//  NOTE: all of below was first way of doing styling, automatically
//  it is somewhat slower than the explicit manual code which is after
//  all not that hard to write -- maintenance may be more of an issue..
//  but given how time-critical styling is, it is worth it overall..

////////////////////////////////////////////////////////////////////////////////////////
//   StyledFields

// StyleFields contain the StyledFields for Style type
var StyleFields = initStyle()

func initStyle() *StyledFields {
	StyleDefault.Defaults()
	sf := &StyledFields{}
	sf.Init(&StyleDefault)
	return sf
}

// StyledFields contains fields of a struct that are styled -- create one
// instance of this for each type that has styled fields (Style, Paint, and a
// few with ad-hoc styled fields)
type StyledFields struct {
	Fields   map[string]*StyledField `desc:"the compiled stylable fields, mapped for the xml and alt tags for the field"`
	Inherits []*StyledField          `desc:"the compiled stylable fields that have inherit:true tags and should thus be inherited from parent objects"`
	Units    []*StyledField          `desc:"the compiled stylable fields of the unit.Value type, which should have ToDots run on them"`
	Default  interface{}             `desc:"points to the Default instance of this type, initialized with the default values used for 'initial' keyword"`
}

func (sf *StyledFields) Init(df interface{}) {
	sf.Default = df
	sf.CompileFields(df)
}

// get the full effective tag based on outer tag plus given tag
func StyleEffTag(tag, outerTag string) string {
	tagEff := tag
	if outerTag != "" && len(tag) > 0 {
		if tag[0] == '.' {
			tagEff = outerTag + tag
		} else {
			tagEff = outerTag + "-" + tag
		}
	}
	return tagEff
}

// AddField adds a single field -- must be a direct field on the object and
// not a field on an embedded type -- used for Widget objects where only one
// or a few fields are styled
func (sf *StyledFields) AddField(df interface{}, fieldName string) error {
	valtyp := reflect.TypeOf(units.Value{})

	if sf.Fields == nil {
		sf.Fields = make(map[string]*StyledField, 5)
		sf.Inherits = make([]*StyledField, 0, 5)
		sf.Units = make([]*StyledField, 0, 5)
	}
	otp := reflect.TypeOf(df)
	if otp.Kind() != reflect.Ptr {
		err := fmt.Errorf("gi.StyleFields.AddField: must pass pointers to the structs, not type: %v kind %v\n", otp, otp.Kind())
		log.Print(err)
		return err
	}
	ot := otp.Elem()
	if ot.Kind() != reflect.Struct {
		err := fmt.Errorf("gi.StyleFields.AddField: only works on structs, not type: %v kind %v\n", ot, ot.Kind())
		log.Print(err)
		return err
	}
	vo := reflect.ValueOf(df).Elem()
	struf, ok := ot.FieldByName(fieldName)
	if !ok {
		err := fmt.Errorf("gi.StyleFields.AddField: field name: %v not found in type %v\n", fieldName, ot.Name())
		log.Print(err)
		return err
	}

	vf := vo.FieldByName(fieldName)

	styf := &StyledField{Field: struf, NetOff: struf.Offset, Default: vf}
	tag := struf.Tag.Get("xml")
	sf.Fields[tag] = styf
	atags := struf.Tag.Get("alt")
	if atags != "" {
		atag := strings.Split(atags, ",")

		for _, tg := range atag {
			sf.Fields[tg] = styf
		}
	}
	inhs := struf.Tag.Get("inherit")
	if inhs == "true" {
		sf.Inherits = append(sf.Inherits, styf)
	}
	if vf.Kind() == reflect.Struct && vf.Type() == valtyp {
		sf.Units = append(sf.Units, styf)
	}
	return nil
}

// CompileFields gathers all the fields with xml tag != "-", plus those
// that are units.Value's for later optimized processing of styles
func (sf *StyledFields) CompileFields(df interface{}) {
	valtyp := reflect.TypeOf(units.Value{})

	sf.Fields = make(map[string]*StyledField, 50)
	sf.Inherits = make([]*StyledField, 0, 50)
	sf.Units = make([]*StyledField, 0, 50)

	WalkStyleStruct(df, "", uintptr(0),
		func(struf reflect.StructField, vf reflect.Value, outerTag string, baseoff uintptr) {
			styf := &StyledField{Field: struf, NetOff: baseoff + struf.Offset, Default: vf}
			tag := StyleEffTag(struf.Tag.Get("xml"), outerTag)
			if _, ok := sf.Fields[tag]; ok {
				fmt.Printf("gi.StyledFileds.CompileFields: ERROR redundant tag found -- please only use unique tags! %v\n", tag)
			}
			sf.Fields[tag] = styf
			atags := struf.Tag.Get("alt")
			if atags != "" {
				atag := strings.Split(atags, ",")

				for _, tg := range atag {
					tag = StyleEffTag(tg, outerTag)
					sf.Fields[tag] = styf
				}
			}
			inhs := struf.Tag.Get("inherit")
			if inhs == "true" {
				sf.Inherits = append(sf.Inherits, styf)
			}
			if vf.Kind() == reflect.Struct && vf.Type() == valtyp {
				sf.Units = append(sf.Units, styf)
			}
		})
	return
}

// Inherit copies all the values from par to obj for fields marked as
// "inherit" -- inherited by default.  NOTE: No longer using this -- doing it
// manually -- much faster
func (sf *StyledFields) Inherit(obj, par interface{}, vp *Viewport2D) {
	// pr := prof.Start("StyleFields.Inherit")
	objptr := reflect.ValueOf(obj).Pointer()
	hasPar := !kit.IfaceIsNil(par)
	var parptr uintptr
	if hasPar {
		parptr = reflect.ValueOf(par).Pointer()
	}
	for _, fld := range sf.Inherits {
		pfi := fld.FieldIface(parptr)
		fld.FromProps(sf.Fields, objptr, parptr, pfi, hasPar, vp)
		// fmt.Printf("inh: %v\n", fld.Field.Name)
	}
	// pr.End()
}

// Style applies styles to the fields from given properties for given object
func (sf *StyledFields) Style(obj, par interface{}, props ki.Props, vp *Viewport2D) {
	if props == nil {
		return
	}
	pr := prof.Start("StyleFields.Style")
	objptr := reflect.ValueOf(obj).Pointer()
	hasPar := !kit.IfaceIsNil(par)
	var parptr uintptr
	if hasPar {
		parptr = reflect.ValueOf(par).Pointer()
	}
	// fewer props than fields, esp with alts!
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		if vstr, ok := val.(string); ok {
			if len(vstr) > 0 && vstr[0] == '$' { // special case to use other value
				nkey := vstr[1:] // e.g., border-color has "$background-color" value
				if vfld, nok := sf.Fields[nkey]; nok {
					nval := vfld.FieldIface(objptr)
					if fld, fok := sf.Fields[key]; fok {
						fld.FromProps(sf.Fields, objptr, parptr, nval, hasPar, vp)
						continue
					}
				}
				fmt.Printf("gi.StyledFields.Style: redirect field not found: %v for key: %v\n", nkey, key)
			}
		}
		fld, ok := sf.Fields[key]
		if !ok {
			// note: props can apply to Paint or Style and not easy to keep those
			// precisely separated, so there will be mismatch..
			// log.Printf("SetStyleFields: Property key: %v not among xml or alt field tags for styled obj: %T\n", key, obj)
			continue
		}
		fld.FromProps(sf.Fields, objptr, parptr, val, hasPar, vp)
	}
	pr.End()
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (sf *StyledFields) ToDots(obj interface{}, uc *units.Context) {
	// pr := prof.Start("StyleFields.ToDots")
	objptr := reflect.ValueOf(obj).Pointer()
	for _, fld := range sf.Units {
		uv := fld.UnitsValue(objptr)
		uv.ToDots(uc)
	}
	// pr.End()
}

////////////////////////////////////////////////////////////////////////////////////////
//   StyledField

// StyledField contains the relevant data for a given stylable field in a struct
type StyledField struct {
	Field   reflect.StructField
	NetOff  uintptr       `desc:"net accumulated offset from the overall main type, e.g., Style"`
	Default reflect.Value `desc:"value of default value of this field"`
}

// FieldValue returns a reflect.Value for a given object, computed from NetOff
// -- this is VERY expensive time-wise -- need to figure out a better solution..
func (sf *StyledField) FieldValue(objptr uintptr) reflect.Value {
	f := unsafe.Pointer(objptr + sf.NetOff)
	nw := reflect.NewAt(sf.Field.Type, f)
	return kit.UnhideIfaceValue(nw).Elem()
}

// FieldIface returns an interface{} for a given object, computed from NetOff
// -- much faster -- use this
func (sf *StyledField) FieldIface(objptr uintptr) interface{} {
	npt := kit.NonPtrType(sf.Field.Type)
	npk := npt.Kind()
	switch {
	case npt == KiT_Color:
		return (*Color)(unsafe.Pointer(objptr + sf.NetOff))
	case npt == KiT_ColorSpec:
		return (*ColorSpec)(unsafe.Pointer(objptr + sf.NetOff))
	case npt == KiT_Matrix2D:
		return (*Matrix2D)(unsafe.Pointer(objptr + sf.NetOff))
	case npt.Name() == "Value":
		return (*units.Value)(unsafe.Pointer(objptr + sf.NetOff))
	case npk >= reflect.Int && npk <= reflect.Uint64:
		return sf.FieldValue(objptr).Interface() // no choice for enums
	case npk == reflect.Float32:
		return (*float32)(unsafe.Pointer(objptr + sf.NetOff))
	case npk == reflect.Float64:
		return (*float64)(unsafe.Pointer(objptr + sf.NetOff))
	case npk == reflect.Bool:
		return (*bool)(unsafe.Pointer(objptr + sf.NetOff))
	case npk == reflect.String:
		return (*string)(unsafe.Pointer(objptr + sf.NetOff))
	case sf.Field.Name == "Dashes":
		return (*[]float64)(unsafe.Pointer(objptr + sf.NetOff))
	default:
		fmt.Printf("Field: %v type %v not processed in StyledField.FieldIface -- fixme!\n", sf.Field.Name, npt.String())
		return nil
	}
}

// UnitsValue returns a units.Value for a field, which must be of that type..
func (sf *StyledField) UnitsValue(objptr uintptr) *units.Value {
	uv := (*units.Value)(unsafe.Pointer(objptr + sf.NetOff))
	return uv
}

// FromProps styles given field from property value val, with optional parent object obj
func (fld *StyledField) FromProps(fields map[string]*StyledField, objptr, parptr uintptr, val interface{}, hasPar bool, vp *Viewport2D) {
	errstr := "gi.StyledField FromProps: Field:"
	fi := fld.FieldIface(objptr)
	if kit.IfaceIsNil(fi) {
		fmt.Printf("%v %v of type %v has nil value\n", errstr, fld.Field.Name, fld.Field.Type.String())
		return
	}
	switch valv := val.(type) {
	case string:
		if valv == "inherit" {
			if hasPar {
				val = fld.FieldIface(parptr)
				// fmt.Printf("%v %v set to inherited value: %v\n", errstr, fld.Field.Name, val)
			} else {
				// fmt.Printf("%v %v tried to inherit but par null: %v\n", errstr, fld.Field.Name, val)
				return
			}
		}
		if valv == "initial" {
			val = fld.Default.Interface()
			// fmt.Printf("%v set tag: %v to initial default value: %v\n", errstr, tag, df)
		}
	}
	// todo: support keywords such as auto, normal, which should just set to 0

	npt := kit.NonPtrType(reflect.TypeOf(fi))
	npk := npt.Kind()

	switch fiv := fi.(type) {
	case *ColorSpec:
		fiv.SetIFace(val, vp, fld.Field.Name)
	case *Color:
		fiv.SetIFace(val, vp, fld.Field.Name)
	case *units.Value:
		fiv.SetIFace(val, fld.Field.Name)
	case *Matrix2D:
		switch valv := val.(type) {
		case string:
			fiv.SetString(valv)
		case *Matrix2D:
			*fiv = *valv
		}
	case *[]float64:
		switch valv := val.(type) {
		case string:
			*fiv = ParseDashesString(valv)
		case *[]float64:
			*fiv = *valv
		}
	default:
		if npk >= reflect.Int && npk <= reflect.Uint64 {
			switch valv := val.(type) {
			case string:
				tn := kit.ShortTypeName(fld.Field.Type)
				if kit.Enums.Enum(tn) != nil {
					kit.Enums.SetAnyEnumIfaceFromString(fi, valv)
				} else if tn == "..int" {
					kit.SetRobust(fi, val)
				} else {
					fmt.Printf("%v enum name not found %v for field %v\n", errstr, tn, fld.Field.Name)
				}
			default:
				ival, ok := kit.ToInt(val)
				if !ok {
					log.Printf("%v for field: %v could not convert property to int: %v %T\n", errstr, fld.Field.Name, val, val)
				} else {
					kit.SetEnumIfaceFromInt64(fi, ival, npt)
				}
			}
		} else {
			kit.SetRobust(fi, val)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//   WalkStyleStruct

// this is the function to process a given field when walking the style
type WalkStyleFieldFunc func(struf reflect.StructField, vf reflect.Value, tag string, baseoff uintptr)

// StyleValueTypes is a map of types that are used as value types in style structures
var StyleValueTypes = map[reflect.Type]struct{}{
	units.KiT_Value: {},
	KiT_Color:       {},
	KiT_ColorSpec:   {},
	KiT_Matrix2D:    {},
}

// WalkStyleStruct walks through a struct, calling a function on fields with
// xml tags that are not "-", recursively through all the fields
func WalkStyleStruct(obj interface{}, outerTag string, baseoff uintptr, fun WalkStyleFieldFunc) {
	otp := reflect.TypeOf(obj)
	if otp.Kind() != reflect.Ptr {
		log.Printf("gi.WalkStyleStruct -- you must pass pointers to the structs, not type: %v kind %v\n", otp, otp.Kind())
		return
	}
	ot := otp.Elem()
	if ot.Kind() != reflect.Struct {
		log.Printf("gi.WalkStyleStruct -- only works on structs, not type: %v kind %v\n", ot, ot.Kind())
		return
	}
	vo := reflect.ValueOf(obj).Elem()
	for i := 0; i < ot.NumField(); i++ {
		struf := ot.Field(i)
		if struf.PkgPath != "" { // skip unexported fields
			continue
		}
		tag := struf.Tag.Get("xml")
		if tag == "-" {
			continue
		}
		ft := struf.Type
		// note: need Addrs() to pass pointers to fields, not fields themselves
		// fmt.Printf("processing field named: %v\n", struf.Nm)
		vf := vo.Field(i)
		vfi := vf.Addr().Interface()
		_, styvaltype := StyleValueTypes[ft]
		if ft.Kind() == reflect.Struct && !styvaltype {
			WalkStyleStruct(vfi, tag, baseoff+struf.Offset, fun)
		} else {
			if tag == "" { // non-struct = don't process
				continue
			}
			fun(struf, vf, outerTag, baseoff)
		}
	}
}
