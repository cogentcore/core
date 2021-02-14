// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"log"

	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// These functions set styles from ki.Props which are used for styling

// StyleInhInit detects the style values of "inherit" and "initial",
// setting the corresponding bool return values
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
	log.Printf("gist.Style error: cannot set key: %s from value: %v\n", key, val)
}

type StyleFunc func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context)

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
				sfunc(&s.Layout, key, val, &par.Layout, ctxt)
			} else {
				sfunc(&s.Layout, key, val, nil, ctxt)
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
		if sfunc, ok := StyleShadowFuncs[key]; ok {
			if par != nil {
				sfunc(&s.BoxShadow, key, val, &par.BoxShadow, ctxt)
			} else {
				sfunc(&s.BoxShadow, key, val, nil, ctxt)
			}
			continue
		}
	}
}

/////////////////////////////////////////////////////////////////////////////////
//  Style

// StyleStyleFuncs are functions for styling the Style object itself
var StyleStyleFuncs = map[string]StyleFunc{
	"display": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"visible": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"inactive": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"pointer-events": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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

/////////////////////////////////////////////////////////////////////////////////
//  Layout

// StyleLayoutFuncs are functions for styling the Layout object
var StyleLayoutFuncs = map[string]StyleFunc{
	"z-index": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.ZIndex = par.(*Layout).ZIndex
			} else if init {
				ly.ZIndex = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ly.ZIndex = int(iv)
		}
	},
	"horizontal-align": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.AlignH = par.(*Layout).AlignH
			} else if init {
				ly.AlignH = AlignLeft
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&ly.AlignH, vt)
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
	"vertical-align": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.AlignV = par.(*Layout).AlignV
			} else if init {
				ly.AlignV = AlignMiddle
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&ly.AlignV, vt)
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
	"x": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.PosX = par.(*Layout).PosX
			} else if init {
				ly.PosX.Val = 0
			}
			return
		}
		ly.PosX.SetIFace(val, key)
	},
	"y": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.PosY = par.(*Layout).PosY
			} else if init {
				ly.PosY.Val = 0
			}
			return
		}
		ly.PosY.SetIFace(val, key)
	},
	"width": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Width = par.(*Layout).Width
			} else if init {
				ly.Width.Val = 0
			}
			return
		}
		ly.Width.SetIFace(val, key)
	},
	"height": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Height = par.(*Layout).Height
			} else if init {
				ly.Height.Val = 0
			}
			return
		}
		ly.Height.SetIFace(val, key)
	},
	"max-width": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.MaxWidth = par.(*Layout).MaxWidth
			} else if init {
				ly.MaxWidth.Val = 0
			}
			return
		}
		ly.MaxWidth.SetIFace(val, key)
	},
	"max-height": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.MaxHeight = par.(*Layout).MaxHeight
			} else if init {
				ly.MaxHeight.Val = 0
			}
			return
		}
		ly.MaxHeight.SetIFace(val, key)
	},
	"min-width": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.MinWidth = par.(*Layout).MinWidth
			} else if init {
				ly.MinWidth.Set(2, units.Px)
			}
			return
		}
		ly.MinWidth.SetIFace(val, key)
	},
	"min-height": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.MinHeight = par.(*Layout).MinHeight
			} else if init {
				ly.MinHeight.Set(2, units.Px)
			}
			return
		}
		ly.MinHeight.SetIFace(val, key)
	},
	"margin": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Margin = par.(*Layout).Margin
			} else if init {
				ly.Margin.Val = 0
			}
			return
		}
		ly.Margin.SetIFace(val, key)
	},
	"padding": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Padding = par.(*Layout).Padding
			} else if init {
				ly.Padding.Val = 0
			}
			return
		}
		ly.Padding.SetIFace(val, key)
	},
	"overflow": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Overflow = par.(*Layout).Overflow
			} else if init {
				ly.Overflow = OverflowAuto
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&ly.Overflow, vt)
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
	"columns": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Columns = par.(*Layout).Columns
			} else if init {
				ly.Columns = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ly.Columns = int(iv)
		}
	},
	"row": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Row = par.(*Layout).Row
			} else if init {
				ly.Row = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ly.Row = int(iv)
		}
	},
	"col": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.Col = par.(*Layout).Col
			} else if init {
				ly.Col = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ly.Col = int(iv)
		}
	},
	"row-span": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.RowSpan = par.(*Layout).RowSpan
			} else if init {
				ly.RowSpan = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ly.RowSpan = int(iv)
		}
	},
	"col-span": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.ColSpan = par.(*Layout).ColSpan
			} else if init {
				ly.ColSpan = 0
			}
			return
		}
		if iv, ok := kit.ToInt(val); ok {
			ly.ColSpan = int(iv)
		}
	},
	"scrollbar-width": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ly := obj.(*Layout)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ly.ScrollBarWidth = par.(*Layout).ScrollBarWidth
			} else if init {
				ly.ScrollBarWidth.Val = 0
			}
			return
		}
		ly.ScrollBarWidth.SetIFace(val, key)
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Font

// StyleFontFuncs are functions for styling the Font object
var StyleFontFuncs = map[string]StyleFunc{
	"color": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*Font).Color
			} else if init {
				fs.Color = Black
			}
			return
		}
		fs.Color.SetIFace(val, ctxt, key)
	},
	"background-color": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.BgColor = par.(*Font).BgColor
			} else if init {
				fs.BgColor = ColorSpec{}
			}
			return
		}
		fs.BgColor.SetIFace(val, ctxt, key)
	},
	"opacity": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"font-size": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Size = par.(*Font).Size
			} else if init {
				fs.Size.Set(12, units.Pt)
			}
			return
		}
		switch vt := val.(type) {
		case string:
			if psz, ok := FontSizePoints[vt]; ok {
				fs.Size = units.NewPt(psz)
			} else {
				fs.Size.SetIFace(val, key) // also processes string
			}
		default:
			fs.Size.SetIFace(val, key)
		}
	},
	"font-family": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"font-style": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"font-weight": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"font-stretch": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"font-variant": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"text-decoration": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"baseline-shift": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"text-align": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"text-vertical-align": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"text-anchor": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"letter-spacing": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"word-spacing": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"line-height": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		ts := obj.(*Text)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ts.LineHeight = par.(*Text).LineHeight
			} else if init {
				ts.LineHeight = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			ts.LineHeight = iv
		}
	},
	"white-space": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"unicode-bidi": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"direction": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"writing-mode": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"glyph-orientation-vertical": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"glyph-orientation-horizontal": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"text-indent": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"para-spacing": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"tab-size": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"border-style": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Style = par.(*Border).Style
			} else if init {
				bs.Style = BorderSolid
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&bs.Style, vt)
		case BorderStyles:
			bs.Style = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				bs.Style = BorderStyles(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"border-width": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Width = par.(*Border).Width
			} else if init {
				bs.Width.Val = 0
			}
			return
		}
		bs.Width.SetIFace(val, key)
	},
	"border-radius": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Radius = par.(*Border).Radius
			} else if init {
				bs.Radius.Val = 0
			}
			return
		}
		bs.Radius.SetIFace(val, key)
	},
	"border-color": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Color = par.(*Border).Color
			} else if init {
				bs.Color = Black
			}
			return
		}
		bs.Color.SetIFace(val, ctxt, key)
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Outline

// StyleOutlineFuncs are functions for styling the OutlineStyle object
var StyleOutlineFuncs = map[string]StyleFunc{
	"outline-style": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Style = par.(*Border).Style
			} else if init {
				bs.Style = BorderNone
			}
			return
		}
		switch vt := val.(type) {
		case string:
			kit.Enums.SetAnyEnumIfaceFromString(&bs.Style, vt)
		case BorderStyles:
			bs.Style = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				bs.Style = BorderStyles(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"outline-width": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Width = par.(*Border).Width
			} else if init {
				bs.Width.Val = 0
			}
			return
		}
		bs.Width.SetIFace(val, key)
	},
	"outline-radius": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Radius = par.(*Border).Radius
			} else if init {
				bs.Radius.Val = 0
			}
			return
		}
		bs.Radius.SetIFace(val, key)
	},
	"outline-color": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Color = par.(*Border).Color
			} else if init {
				bs.Color = Black
			}
			return
		}
		bs.Color.SetIFace(val, ctxt, key)
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Shadow

// StyleShadowFuncs are functions for styling the Shadow object
var StyleShadowFuncs = map[string]StyleFunc{
	"box-shadow.h-offset": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"box-shadow.v-offset": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"box-shadow.blur": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"box-shadow.spread": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"box-shadow.color": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"box-shadow.inset": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
