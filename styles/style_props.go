// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"log/slog"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/units"
	"cogentcore.org/core/xgo/num"
)

// StyleInhInit detects the style values of "inherit" and "initial",
// setting the corresponding bool return values
func StyleInhInit(val, parent any) (inh, init bool) {
	if str, ok := val.(string); ok {
		switch str {
		case "inherit":
			return !laser.AnyIsNil(parent), false
		case "initial":
			return false, true
		default:
			return false, false
		}
	}
	return false, false
}

// StyleFuncInt returns a style function for any numerical value
func StyleFuncInt[T any, F num.Integer](initVal F, getField func(obj *T) *F) StyleFunc {
	return func(obj any, key string, val any, parent any, cc colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				*fp = *getField(parent.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		fv, _ := laser.ToInt(val)
		num.SetNumber(fp, fv)
	}
}

// StyleFuncFloat returns a style function for any numerical value
func StyleFuncFloat[T any, F num.Float](initVal F, getField func(obj *T) *F) StyleFunc {
	return func(obj any, key string, val any, parent any, cc colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				*fp = *getField(parent.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		fv, _ := laser.ToFloat(val) // can represent any number, ToFloat is fast type switch
		num.SetNumber(fp, fv)
	}
}

// StyleFuncBool returns a style function for a bool value
func StyleFuncBool[T any](initVal bool, getField func(obj *T) *bool) StyleFunc {
	return func(obj any, key string, val any, parent any, cc colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				*fp = *getField(parent.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		fv, _ := laser.ToBool(val)
		*fp = fv
	}
}

// StyleFuncUnits returns a style function for units.Value
func StyleFuncUnits[T any](initVal units.Value, getField func(obj *T) *units.Value) StyleFunc {
	return func(obj any, key string, val any, parent any, cc colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				*fp = *getField(parent.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		fp.SetAny(val, key)
	}
}

// StyleFuncEnum returns a style function for any enum value
func StyleFuncEnum[T any](initVal enums.Enum, getField func(obj *T) enums.EnumSetter) StyleFunc {
	return func(obj any, key string, val any, parent any, cc colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				fp.SetInt64(getField(parent.(*T)).Int64())
			} else if init {
				fp.SetInt64(initVal.Int64())
			}
			return
		}
		if st, ok := val.(string); ok {
			fp.SetString(st)
			return
		}
		if en, ok := val.(enums.Enum); ok {
			fp.SetInt64(en.Int64())
			return
		}
		iv, _ := laser.ToInt(val)
		fp.SetInt64(int64(iv))
	}
}

// These functions set styles from map[string]any which are used for styling

// StyleSetError reports that cannot set property of given key with given value due to given error
func StyleSetError(key string, val any, err error) {
	slog.Error("styles.Style: error setting value", "key", key, "value", val, "err", err)
}

type StyleFunc func(obj any, key string, val any, parent any, cc colors.Context)

// StyleFromProp sets style field values based on the given property key and value
func (s *Style) StyleFromProp(parent *Style, key string, val any, cc colors.Context) {
	if sfunc, ok := StyleLayoutFuncs[key]; ok {
		if parent != nil {
			sfunc(s, key, val, parent, cc)
		} else {
			sfunc(s, key, val, nil, cc)
		}
		return
	}
	if sfunc, ok := StyleFontFuncs[key]; ok {
		if parent != nil {
			sfunc(&s.Font, key, val, &parent.Font, cc)
		} else {
			sfunc(&s.Font, key, val, nil, cc)
		}
		return
	}
	if sfunc, ok := StyleTextFuncs[key]; ok {
		if parent != nil {
			sfunc(&s.Text, key, val, &parent.Text, cc)
		} else {
			sfunc(&s.Text, key, val, nil, cc)
		}
		return
	}
	if sfunc, ok := StyleBorderFuncs[key]; ok {
		if parent != nil {
			sfunc(&s.Border, key, val, &parent.Border, cc)
		} else {
			sfunc(&s.Border, key, val, nil, cc)
		}
		return
	}
	if sfunc, ok := StyleStyleFuncs[key]; ok {
		sfunc(s, key, val, parent, cc)
		return
	}
	// doesn't work with multiple shadows
	// if sfunc, ok := StyleShadowFuncs[key]; ok {
	// 	if parent != nil {
	// 		sfunc(&s.BoxShadow, key, val, &par.BoxShadow, cc)
	// 	} else {
	// 		sfunc(&s.BoxShadow, key, val, nil, cc)
	// 	}
	// 	return
	// }
}

/////////////////////////////////////////////////////////////////////////////////
//  Style

// StyleStyleFuncs are functions for styling the Style object itself
var StyleStyleFuncs = map[string]StyleFunc{
	"color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				fs.Color = parent.(*Style).Color
			} else if init {
				fs.Color = colors.C(colors.Scheme.OnSurface)
			}
			return
		}
		fs.Color = grr.Log1(gradient.FromAny(val, cc))
	},
	"background-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				fs.Background = parent.(*Style).Background
			} else if init {
				fs.Background = nil
			}
			return
		}
		fs.Background = grr.Log1(gradient.FromAny(val, cc))
	},
	"opacity": StyleFuncFloat(float32(1),
		func(obj *Style) *float32 { return &obj.Opacity }),
}

/////////////////////////////////////////////////////////////////////////////////
//  Layout

// StyleLayoutFuncs are functions for styling the layout
// style properties; they are still stored on the main style object,
// but they are done separately to improve clarity
var StyleLayoutFuncs = map[string]StyleFunc{
	"display": StyleFuncEnum(Flex,
		func(obj *Style) enums.EnumSetter { return &obj.Display }),
	"flex-direction": func(obj any, key string, val, parent any, cc colors.Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				s.Direction = parent.(*Style).Direction
			} else if init {
				s.Direction = Row
			}
			return
		}
		str := laser.ToString(val)
		if str == "row" || str == "row-reverse" {
			s.Direction = Row
		} else {
			s.Direction = Column
		}
	},
	// TODO(kai/styproperties): multi-dim flex-grow
	"flex-grow": StyleFuncFloat(0, func(obj *Style) *float32 { return &obj.Grow.Y }),
	"wrap": StyleFuncBool(false,
		func(obj *Style) *bool { return &obj.Wrap }),
	"justify-content": StyleFuncEnum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Justify.Content }),
	"justify-items": StyleFuncEnum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Justify.Items }),
	"justify-self": StyleFuncEnum(Auto,
		func(obj *Style) enums.EnumSetter { return &obj.Justify.Self }),
	"align-content": StyleFuncEnum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Align.Content }),
	"align-items": StyleFuncEnum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Align.Items }),
	"align-self": StyleFuncEnum(Auto,
		func(obj *Style) enums.EnumSetter { return &obj.Align.Self }),
	"x": StyleFuncUnits(units.Value{},
		func(obj *Style) *units.Value { return &obj.Pos.X }),
	"y": StyleFuncUnits(units.Value{},
		func(obj *Style) *units.Value { return &obj.Pos.Y }),
	"width": StyleFuncUnits(units.Value{},
		func(obj *Style) *units.Value { return &obj.Min.X }),
	"height": StyleFuncUnits(units.Value{},
		func(obj *Style) *units.Value { return &obj.Min.Y }),
	"max-width": StyleFuncUnits(units.Value{},
		func(obj *Style) *units.Value { return &obj.Max.X }),
	"max-height": StyleFuncUnits(units.Value{},
		func(obj *Style) *units.Value { return &obj.Max.Y }),
	"min-width": StyleFuncUnits(units.Dp(2),
		func(obj *Style) *units.Value { return &obj.Min.X }),
	"min-height": StyleFuncUnits(units.Dp(2),
		func(obj *Style) *units.Value { return &obj.Min.Y }),
	"margin": func(obj any, key string, val any, parent any, cc colors.Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				s.Margin = parent.(*Style).Margin
			} else if init {
				s.Margin.Zero()
			}
			return
		}
		s.Margin.SetAny(val)
	},
	"padding": func(obj any, key string, val any, parent any, cc colors.Context) {
		s := obj.(*Style)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				s.Padding = parent.(*Style).Padding
			} else if init {
				s.Padding.Zero()
			}
			return
		}
		s.Padding.SetAny(val)
	},
	// TODO(kai/styproperties): multi-dim overflow
	"overflow": StyleFuncEnum(OverflowAuto,
		func(obj *Style) enums.EnumSetter { return &obj.Overflow.Y }),
	"columns": StyleFuncInt(int(0),
		func(obj *Style) *int { return &obj.Columns }),
	"row": StyleFuncInt(int(0),
		func(obj *Style) *int { return &obj.Row }),
	"col": StyleFuncInt(int(0),
		func(obj *Style) *int { return &obj.Col }),
	"row-span": StyleFuncInt(int(0),
		func(obj *Style) *int { return &obj.RowSpan }),
	"col-span": StyleFuncInt(int(0),
		func(obj *Style) *int { return &obj.ColSpan }),
	"z-index": StyleFuncInt(int(0),
		func(obj *Style) *int { return &obj.ZIndex }),
	"scrollbar-width": StyleFuncUnits(units.Value{},
		func(obj *Style) *units.Value { return &obj.ScrollBarWidth }),
}

/////////////////////////////////////////////////////////////////////////////////
//  Font

// StyleFontFuncs are functions for styling the Font object
var StyleFontFuncs = map[string]StyleFunc{
	"font-size": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				fs.Size = parent.(*Font).Size
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
				fs.Size.SetAny(val, key) // also processes string
			}
		default:
			fs.Size.SetAny(val, key)
		}
	},
	"font-family": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				fs.Family = parent.(*Font).Family
			} else if init {
				fs.Family = "" // font has defaults
			}
			return
		}
		fs.Family = laser.ToString(val)
	},
	"font-style": StyleFuncEnum(FontNormal,
		func(obj *Font) enums.EnumSetter { return &obj.Style }),
	"font-weight": StyleFuncEnum(WeightNormal,
		func(obj *Font) enums.EnumSetter { return &obj.Weight }),
	"font-stretch": StyleFuncEnum(FontStrNormal,
		func(obj *Font) enums.EnumSetter { return &obj.Stretch }),
	"font-variant": StyleFuncEnum(FontVarNormal,
		func(obj *Font) enums.EnumSetter { return &obj.Variant }),
	"baseline-shift": StyleFuncEnum(ShiftBaseline,
		func(obj *Font) enums.EnumSetter { return &obj.Shift }),
	"text-decoration": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				fs.Decoration = parent.(*Font).Decoration
			} else if init {
				fs.Decoration = DecoNone
			}
			return
		}
		switch vt := val.(type) {
		case string:
			if vt == "none" {
				fs.Decoration = DecoNone
			} else {
				fs.Decoration.SetString(vt)
			}
		case TextDecorations:
			fs.Decoration = vt
		default:
			iv, err := laser.ToInt(val)
			if err == nil {
				fs.Decoration = TextDecorations(iv)
			} else {
				StyleSetError(key, val, err)
			}
		}
	},
}

// StyleFontRenderFuncs are _extra_ functions for styling
// the FontRender object in addition to base Font
var StyleFontRenderFuncs = map[string]StyleFunc{
	"color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*FontRender)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				fs.Color = parent.(*FontRender).Color
			} else if init {
				fs.Color = colors.C(colors.Scheme.OnSurface)
			}
			return
		}
		fs.Color = grr.Log1(gradient.FromAny(val, cc))
	},
	"background-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*FontRender)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				fs.Background = parent.(*FontRender).Background
			} else if init {
				fs.Background = nil
			}
			return
		}
		fs.Background = grr.Log1(gradient.FromAny(val, cc))
	},
	"opacity": StyleFuncFloat(float32(1),
		func(obj *FontRender) *float32 { return &obj.Opacity }),
}

/////////////////////////////////////////////////////////////////////////////////
//  Text

// StyleTextFuncs are functions for styling the Text object
var StyleTextFuncs = map[string]StyleFunc{
	"text-align": StyleFuncEnum(Start,
		func(obj *Text) enums.EnumSetter { return &obj.Align }),
	"text-vertical-align": StyleFuncEnum(Start,
		func(obj *Text) enums.EnumSetter { return &obj.AlignV }),
	"text-anchor": StyleFuncEnum(AnchorStart,
		func(obj *Text) enums.EnumSetter { return &obj.Anchor }),
	"letter-spacing": StyleFuncUnits(units.Value{},
		func(obj *Text) *units.Value { return &obj.LetterSpacing }),
	"word-spacing": StyleFuncUnits(units.Value{},
		func(obj *Text) *units.Value { return &obj.WordSpacing }),
	"line-height": StyleFuncUnits(LineHeightNormal,
		func(obj *Text) *units.Value { return &obj.LineHeight }),
	"white-space": StyleFuncEnum(WhiteSpaceNormal,
		func(obj *Text) enums.EnumSetter { return &obj.WhiteSpace }),
	"unicode-bidi": StyleFuncEnum(BidiNormal,
		func(obj *Text) enums.EnumSetter { return &obj.UnicodeBidi }),
	"direction": StyleFuncEnum(LRTB,
		func(obj *Text) enums.EnumSetter { return &obj.Direction }),
	"writing-mode": StyleFuncEnum(LRTB,
		func(obj *Text) enums.EnumSetter { return &obj.WritingMode }),
	"glyph-orientation-vertical": StyleFuncFloat(float32(1),
		func(obj *Text) *float32 { return &obj.OrientationVert }),
	"glyph-orientation-horizontal": StyleFuncFloat(float32(1),
		func(obj *Text) *float32 { return &obj.OrientationHoriz }),
	"text-indent": StyleFuncUnits(units.Value{},
		func(obj *Text) *units.Value { return &obj.Indent }),
	"para-spacing": StyleFuncUnits(units.Value{},
		func(obj *Text) *units.Value { return &obj.ParaSpacing }),
	"tab-size": StyleFuncInt(int(4),
		func(obj *Text) *int { return &obj.TabSize }),
}

/////////////////////////////////////////////////////////////////////////////////
//  Border

// StyleBorderFuncs are functions for styling the Border object
var StyleBorderFuncs = map[string]StyleFunc{
	// SidesTODO: need to figure out how to get key and context information for side SetAny calls
	// with padding, margin, border, etc
	"border-style": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				bs.Style = parent.(*Border).Style
			} else if init {
				bs.Style.Set(BorderSolid)
			}
			return
		}
		switch vt := val.(type) {
		case string:
			bs.Style.SetString(vt)
		case BorderStyles:
			bs.Style.Set(vt)
		case []BorderStyles:
			bs.Style.Set(vt...)
		default:
			iv, err := laser.ToInt(val)
			if err == nil {
				bs.Style.Set(BorderStyles(iv))
			} else {
				StyleSetError(key, val, err)
			}
		}
	},
	"border-width": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				bs.Width = parent.(*Border).Width
			} else if init {
				bs.Width.Zero()
			}
			return
		}
		bs.Width.SetAny(val)
	},
	"border-radius": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				bs.Radius = parent.(*Border).Radius
			} else if init {
				bs.Radius.Zero()
			}
			return
		}
		bs.Radius.SetAny(val)
	},
	"border-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				bs.Color = parent.(*Border).Color
			} else if init {
				bs.Color.Set(colors.C(colors.Scheme.Outline))
			}
			return
		}
		// TODO(kai): support side-specific border colors
		bs.Color.Set(grr.Log1(gradient.FromAny(val, cc)))
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Outline

// StyleOutlineFuncs are functions for styling the OutlineStyle object
var StyleOutlineFuncs = map[string]StyleFunc{
	"outline-style": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				bs.Style = parent.(*Border).Style
			} else if init {
				bs.Style.Set(BorderSolid)
			}
			return
		}
		switch vt := val.(type) {
		case string:
			bs.Style.SetString(vt)
		case BorderStyles:
			bs.Style.Set(vt)
		case []BorderStyles:
			bs.Style.Set(vt...)
		default:
			iv, err := laser.ToInt(val)
			if err == nil {
				bs.Style.Set(BorderStyles(iv))
			} else {
				StyleSetError(key, val, err)
			}
		}
	},
	"outline-width": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				bs.Width = parent.(*Border).Width
			} else if init {
				bs.Width.Zero()
			}
			return
		}
		bs.Width.SetAny(val)
	},
	"outline-radius": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				bs.Radius = parent.(*Border).Radius
			} else if init {
				bs.Radius.Zero()
			}
			return
		}
		bs.Radius.SetAny(val)
	},
	"outline-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				bs.Color = parent.(*Border).Color
			} else if init {
				bs.Color.Set(colors.C(colors.Scheme.Outline))
			}
			return
		}
		// TODO(kai): support side-specific border colors
		bs.Color.Set(grr.Log1(gradient.FromAny(val, cc)))
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Shadow

// StyleShadowFuncs are functions for styling the Shadow object
var StyleShadowFuncs = map[string]StyleFunc{
	"box-shadow.offset-x": StyleFuncUnits(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.OffsetX }),
	"box-shadow.offset-y": StyleFuncUnits(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.OffsetY }),
	"box-shadow.blur": StyleFuncUnits(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.Blur }),
	"box-shadow.spread": StyleFuncUnits(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.Spread }),
	"box-shadow.color": func(obj any, key string, val any, parent any, cc colors.Context) {
		ss := obj.(*Shadow)
		if inh, init := StyleInhInit(val, parent); inh || init {
			if inh {
				ss.Color = parent.(*Shadow).Color
			} else if init {
				ss.Color = colors.C(colors.Scheme.Shadow)
			}
			return
		}
		ss.Color = grr.Log1(gradient.FromAny(val, cc))
	},
	"box-shadow.inset": StyleFuncBool(false,
		func(obj *Shadow) *bool { return &obj.Inset }),
}
