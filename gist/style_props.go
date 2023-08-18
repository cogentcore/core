// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"log"

	"github.com/goki/colors"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// is:
// label.SetProp("background-color", "blue")
//
// should be:
// label.Style.Background.Color = colors.Blue
// label.ActStyle contains the actual style values, which reflects any properties that
// have been set via CSS or SetProp, and those set in Style which serves as the starting
// point for styling.

// These functions set styles from ki.Props which are used for styling

// StyleInhInit detects the style values of "inherit" and "initial",
// setting the corresponding bool return values
func StyleInhInit(val, par any) (inh, init bool) {
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
func StyleSetError(key string, val any) {
	log.Printf("gist.Style error: cannot set key: %s from value: %v\n", key, val)
}

type StyleFunc func(obj any, key string, val any, par any, ctxt Context)

// StyleFromProps sets style field values based on ki.Props properties
func (s *Style) StyleFromProps(par *Style, props ki.Props, ctxt Context) {
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
				sfunc(s, key, val, par, ctxt)
			} else {
				sfunc(s, key, val, nil, ctxt)
			}
			continue
		}
		if sfunc, ok := StyleFontFuncs[key]; ok {
			if par != nil {
				sfunc(&s.Font, key, val, &par.Font, ctxt)
			} else {
				sfunc(&s.Font, key, val, nil, ctxt)
			}
			continue
		}
		if sfunc, ok := StyleTextFuncs[key]; ok {
			if par != nil {
				sfunc(&s.Text, key, val, &par.Text, ctxt)
			} else {
				sfunc(&s.Text, key, val, nil, ctxt)
			}
			continue
		}
		if sfunc, ok := StyleBorderFuncs[key]; ok {
			if par != nil {
				sfunc(&s.Border, key, val, &par.Border, ctxt)
			} else {
				sfunc(&s.Border, key, val, nil, ctxt)
			}
			continue
		}
		if sfunc, ok := StyleStyleFuncs[key]; ok {
			sfunc(s, key, val, par, ctxt)
			continue
		}
		if sfunc, ok := StyleOutlineFuncs[key]; ok {
			if par != nil {
				sfunc(&s.Outline, key, val, &par.Outline, ctxt)
			} else {
				sfunc(&s.Outline, key, val, nil, ctxt)
			}
			continue
		}
		// doesn't work with multiple shadows
		// if sfunc, ok := StyleShadowFuncs[key]; ok {
		// 	if par != nil {
		// 		sfunc(&s.BoxShadow, key, val, &par.BoxShadow, ctxt)
		// 	} else {
		// 		sfunc(&s.BoxShadow, key, val, nil, ctxt)
		// 	}
		// 	continue
		// }
	}
}

/////////////////////////////////////////////////////////////////////////////////
//  Style

// StyleStyleFuncs are functions for styling the Style object itself
var StyleStyleFuncs = map[string]StyleFunc{
	"display": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.Display = par.(*Style).Display
			} else if init {
				s.Display = true
			}
			return
		}
		if kit.ToString(val) == "none" {
			s.Display = false
		} else {
			if bv, ok := kit.ToBool(val); ok {
				s.Display = bv
			}
		}
	},
	"visible": func(obj any, key string, val any, par any, ctxt Context) {
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
	"inactive": func(obj any, key string, val any, par any, ctxt Context) {
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
	"pointer-events": func(obj any, key string, val any, par any, ctxt Context) {
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
	"color": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*Style).Color
			} else if init {
				fs.Color = colors.Black
			}
			return
		}
		fs.Color = colors.LogFromAny(val, ctxt.ContextColor())
	},
	"background-color": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.BackgroundColor = par.(*Style).BackgroundColor
			} else if init {
				fs.BackgroundColor = ColorSpec{}
			}
			return
		}
		fs.BackgroundColor.SetIFace(val, ctxt, key)
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Layout

// StyleLayoutFuncs are functions for styling the layout
// style properties; they are still stored on the main style object,
// but they are done separately to improve clarity
var StyleLayoutFuncs = map[string]StyleFunc{
	"z-index": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.ZIndex = par.(*Style).ZIndex
			} else if init {
				s.ZIndex = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			s.ZIndex = int(iv)
		}
	},
	"horizontal-align": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.AlignH = par.(*Style).AlignH
			} else if init {
				s.AlignH = AlignLeft
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&s.AlignH, vt)
		case Align:
			s.AlignH = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.AlignH = Align(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"vertical-align": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.AlignV = par.(*Style).AlignV
			} else if init {
				s.AlignV = AlignMiddle
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&s.AlignV, vt)
		case Align:
			s.AlignV = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.AlignV = Align(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"x": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.PosX = par.(*Style).PosX
			} else if init {
				s.PosX.Val = 0
			}
			return
		}
		s.PosX.SetIFace(val, key)
	},
	"y": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.PosY = par.(*Style).PosY
			} else if init {
				s.PosY.Val = 0
			}
			return
		}
		s.PosY.SetIFace(val, key)
	},
	"width": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.Width = par.(*Style).Width
			} else if init {
				s.Width.Val = 0
			}
			return
		}
		s.Width.SetIFace(val, key)
	},
	"height": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.Height = par.(*Style).Height
			} else if init {
				s.Height.Val = 0
			}
			return
		}
		s.Height.SetIFace(val, key)
	},
	"max-width": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.MaxWidth = par.(*Style).MaxWidth
			} else if init {
				s.MaxWidth.Val = 0
			}
			return
		}
		s.MaxWidth.SetIFace(val, key)
	},
	"max-height": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.MaxHeight = par.(*Style).MaxHeight
			} else if init {
				s.MaxHeight.Val = 0
			}
			return
		}
		s.MaxHeight.SetIFace(val, key)
	},
	"min-width": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.MinWidth = par.(*Style).MinWidth
			} else if init {
				s.MinWidth.Set(2, units.UnitPx)
			}
			return
		}
		s.MinWidth.SetIFace(val, key)
	},
	"min-height": func(obj any, key string, val any, par any, ctxt Context) {
		ly := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.MinHeight = par.(*Style).MinHeight
			} else if init {
				ly.MinHeight.Set(2, units.UnitPx)
			}
			return
		}
		ly.MinHeight.SetIFace(val, key)
	},
	"margin": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.Margin = par.(*Style).Margin
			} else if init {
				s.Margin.Set()
			}
			return
		}
		s.Margin.SetAny(val)
	},
	"padding": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.Padding = par.(*Style).Padding
			} else if init {
				s.Padding.Set()
			}
			return
		}
		s.Padding.SetAny(val)
	},
	"overflow": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.Overflow = par.(*Style).Overflow
			} else if init {
				s.Overflow = OverflowAuto
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&s.Overflow, vt)
		case Overflow:
			s.Overflow = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				s.Overflow = Overflow(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"columns": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.Columns = par.(*Style).Columns
			} else if init {
				s.Columns = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			s.Columns = int(iv)
		}
	},
	"row": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.Row = par.(*Style).Row
			} else if init {
				s.Row = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			s.Row = int(iv)
		}
	},
	"col": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.Col = par.(*Style).Col
			} else if init {
				s.Col = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			s.Col = int(iv)
		}
	},
	"row-span": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.RowSpan = par.(*Style).RowSpan
			} else if init {
				s.RowSpan = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			s.RowSpan = int(iv)
		}
	},
	"col-span": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.ColSpan = par.(*Style).ColSpan
			} else if init {
				s.ColSpan = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			s.ColSpan = int(iv)
		}
	},
	"scrollbar-width": func(obj any, key string, val any, par any, ctxt Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				s.ScrollBarWidth = par.(*Style).ScrollBarWidth
			} else if init {
				s.ScrollBarWidth.Val = 0
			}
			return
		}
		s.ScrollBarWidth.SetIFace(val, key)
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Font

// StyleFontFuncs are functions for styling the Font object
var StyleFontFuncs = map[string]StyleFunc{
	"opacity": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Opacity = par.(*Font).Opacity
			} else if init {
				fs.Opacity = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			fs.Opacity = iv
		}
	},
	"font-size": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Size = par.(*Font).Size
			} else if init {
				fs.Size.Set(12, units.UnitPt)
			}
			return
		}
		switch vt := val.(type) {
		case string:
			if psz, ok := FontSizePoints[vt]; ok {
				fs.Size = units.Pt(psz)
			} else {
				fs.Size.SetIFace(val, key) // also processes string
			}
		default:
			fs.Size.SetIFace(val, key)
		}
	},
	"font-family": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Family = par.(*Font).Family
			} else if init {
				fs.Family = "" // font has defaults
			}
			return
		}
		fs.Family = kit.ToString(val)
	},
	"font-style": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Style = par.(*Font).Style
			} else if init {
				fs.Style = FontNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&fs.Style, vt)
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
	"font-weight": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Weight = par.(*Font).Weight
			} else if init {
				fs.Weight = WeightNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&fs.Weight, vt)
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
	"font-stretch": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Stretch = par.(*Font).Stretch
			} else if init {
				fs.Stretch = FontStrNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&fs.Stretch, vt)
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
	"font-variant": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Variant = par.(*Font).Variant
			} else if init {
				fs.Variant = FontVarNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&fs.Variant, vt)
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
	"text-decoration": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Deco = par.(*Font).Deco
			} else if init {
				fs.Deco = DecoNone
			}
			return
		}
		switch vt := val.(type) {
		case string:
			if vt == "none" {
				fs.Deco = DecoNone
			} else {
				kit.Enums.SetAnyEnumIfaceFromString(&fs.Deco, vt)
			}
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
	"baseline-shift": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Shift = par.(*Font).Shift
			} else if init {
				fs.Shift = ShiftBaseline
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&fs.Shift, vt)
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

/////////////////////////////////////////////////////////////////////////////////
//  Text

// StyleTextFuncs are functions for styling the Text object
var StyleTextFuncs = map[string]StyleFunc{
	"text-align": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.Align = par.(*Text).Align
			} else if init {
				ts.Align = AlignLeft
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&ts.Align, vt)
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
	"text-vertical-align": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.AlignV = par.(*Text).Align
			} else if init {
				ts.AlignV = AlignTop
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&ts.AlignV, vt)
		case Align:
			ts.AlignV = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				ts.AlignV = Align(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"text-anchor": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.Anchor = par.(*Text).Anchor
			} else if init {
				ts.Anchor = AnchorStart
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&ts.Anchor, vt)
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
	"letter-spacing": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.LetterSpacing = par.(*Text).LetterSpacing
			} else if init {
				ts.LetterSpacing.Val = 0
			}
			return
		}
		ts.LetterSpacing.SetIFace(val, key)
	},
	"word-spacing": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.WordSpacing = par.(*Text).WordSpacing
			} else if init {
				ts.WordSpacing.Val = 0
			}
			return
		}
		ts.WordSpacing.SetIFace(val, key)
	},
	"line-height": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.LineHeight = par.(*Text).LineHeight
			} else if init {
				ts.LineHeight = LineHeightNormal
			}
			return
		}
		ts.LineHeight.SetIFace(val, key)
	},
	"white-space": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.WhiteSpace = par.(*Text).WhiteSpace
			} else if init {
				ts.WhiteSpace = WhiteSpaceNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&ts.WhiteSpace, vt)
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
	"unicode-bidi": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.UnicodeBidi = par.(*Text).UnicodeBidi
			} else if init {
				ts.UnicodeBidi = BidiNormal
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&ts.UnicodeBidi, vt)
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
	"direction": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.Direction = par.(*Text).Direction
			} else if init {
				ts.Direction = LRTB
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&ts.Direction, vt)
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
	"writing-mode": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.WritingMode = par.(*Text).WritingMode
			} else if init {
				ts.WritingMode = LRTB
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&ts.WritingMode, vt)
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
	"glyph-orientation-vertical": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.OrientationVert = par.(*Text).OrientationVert
			} else if init {
				ts.OrientationVert = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			ts.OrientationVert = iv
		}
	},
	"glyph-orientation-horizontal": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.OrientationHoriz = par.(*Text).OrientationHoriz
			} else if init {
				ts.OrientationHoriz = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			ts.OrientationHoriz = iv
		}
	},
	"text-indent": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.Indent = par.(*Text).Indent
			} else if init {
				ts.Indent.Val = 0
			}
			return
		}
		ts.Indent.SetIFace(val, key)
	},
	"para-spacing": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.ParaSpacing = par.(*Text).ParaSpacing
			} else if init {
				ts.ParaSpacing.Val = 0
			}
			return
		}
		ts.ParaSpacing.SetIFace(val, key)
	},
	"tab-size": func(obj any, key string, val any, par any, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.TabSize = par.(*Text).TabSize
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

/////////////////////////////////////////////////////////////////////////////////
//  Border

// StyleBorderFuncs are functions for styling the Border object
var StyleBorderFuncs = map[string]StyleFunc{
	// SidesTODO: need to figure out how to get key and context information for side SetAny calls
	// with padding, margin, border, etc
	"border-style": func(obj any, key string, val any, par any, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Style = par.(*Border).Style
			} else if init {
				bs.Style.Set(BorderSolid)
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&bs.Style, vt)
		case BorderStyles:
			bs.Style.Set(vt)
		case []BorderStyles:
			bs.Style.Set(vt...)
		default:
			if iv, ok := kit.ToInt(val); ok {
				bs.Style.Set(BorderStyles(iv))
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"border-width": func(obj any, key string, val any, par any, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Width = par.(*Border).Width
			} else if init {
				bs.Width.Set()
			}
			return
		}
		bs.Width.SetAny(val)
	},
	"border-radius": func(obj any, key string, val any, par any, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Radius = par.(*Border).Radius
			} else if init {
				bs.Radius.Set()
			}
			return
		}
		bs.Radius.SetAny(val)
	},
	"border-color": func(obj any, key string, val any, par any, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Color = par.(*Border).Color
			} else if init {
				bs.Color.Set(Black)
			}
			return
		}
		bs.Color.SetAny(val, ctxt)
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Outline

// StyleOutlineFuncs are functions for styling the OutlineStyle object
var StyleOutlineFuncs = map[string]StyleFunc{
	"outline-style": func(obj any, key string, val any, par any, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Style = par.(*Border).Style
			} else if init {
				bs.Style.Set(BorderNone)
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&bs.Style, vt)
		case BorderStyles:
			bs.Style.Set(vt)
		case []BorderStyles:
			bs.Style.Set(vt...)
		default:
			if iv, ok := kit.ToInt(val); ok {
				bs.Style.Set(BorderStyles(iv))
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"outline-width": func(obj any, key string, val any, par any, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Width = par.(*Border).Width
			} else if init {
				bs.Width.Set()
			}
			return
		}
		bs.Width.SetAny(val)
	},
	"outline-radius": func(obj any, key string, val any, par any, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Radius = par.(*Border).Radius
			} else if init {
				bs.Radius.Set()
			}
			return
		}
		bs.Radius.SetAny(val)
	},
	"outline-color": func(obj any, key string, val any, par any, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Color = par.(*Border).Color
			} else if init {
				bs.Color.Set(Black)
			}
			return
		}
		bs.Color.SetAny(val, ctxt)
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Shadow

// StyleShadowFuncs are functions for styling the Shadow object
var StyleShadowFuncs = map[string]StyleFunc{
	"box-shadow.h-offset": func(obj any, key string, val any, par any, ctxt Context) {
		ss := obj.(*Shadow)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.HOffset = par.(*Shadow).HOffset
			} else if init {
				ss.HOffset.Val = 0
			}
			return
		}
		ss.HOffset.SetIFace(val, key)
	},
	"box-shadow.v-offset": func(obj any, key string, val any, par any, ctxt Context) {
		ss := obj.(*Shadow)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.VOffset = par.(*Shadow).VOffset
			} else if init {
				ss.VOffset.Val = 0
			}
			return
		}
		ss.VOffset.SetIFace(val, key)
	},
	"box-shadow.blur": func(obj any, key string, val any, par any, ctxt Context) {
		ss := obj.(*Shadow)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.Blur = par.(*Shadow).Blur
			} else if init {
				ss.Blur.Val = 0
			}
			return
		}
		ss.Blur.SetIFace(val, key)
	},
	"box-shadow.spread": func(obj any, key string, val any, par any, ctxt Context) {
		ss := obj.(*Shadow)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.Spread = par.(*Shadow).Spread
			} else if init {
				ss.Spread.Val = 0
			}
			return
		}
		ss.Spread.SetIFace(val, key)
	},
	"box-shadow.color": func(obj any, key string, val any, par any, ctxt Context) {
		ss := obj.(*Shadow)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.Color = par.(*Shadow).Color
			} else if init {
				ss.Color = Black
			}
			return
		}
		ss.Color.SetIFace(val, ctxt, key)
	},
	"box-shadow.inset": func(obj any, key string, val any, par any, ctxt Context) {
		ss := obj.(*Shadow)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.Inset = par.(*Shadow).Inset
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
