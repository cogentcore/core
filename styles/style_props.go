// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/styles/styleprops"
	"cogentcore.org/core/styles/units"
)

// These functions set styles from map[string]any which are used for styling

// StyleFromProperty sets style field values based on the given property key and value
func (s *Style) StyleFromProperty(parent *Style, key string, val any, cc colors.Context) {
	if sfunc, ok := styleLayoutFuncs[key]; ok {
		if parent != nil {
			sfunc(s, key, val, parent, cc)
		} else {
			sfunc(s, key, val, nil, cc)
		}
		return
	}
	if sfunc, ok := styleFontFuncs[key]; ok {
		if parent != nil {
			sfunc(&s.Font, key, val, &parent.Font, cc)
		} else {
			sfunc(&s.Font, key, val, nil, cc)
		}
		return
	}
	if sfunc, ok := styleTextFuncs[key]; ok {
		if parent != nil {
			sfunc(&s.Text, key, val, &parent.Text, cc)
		} else {
			sfunc(&s.Text, key, val, nil, cc)
		}
		return
	}
	if sfunc, ok := styleBorderFuncs[key]; ok {
		if parent != nil {
			sfunc(&s.Border, key, val, &parent.Border, cc)
		} else {
			sfunc(&s.Border, key, val, nil, cc)
		}
		return
	}
	if sfunc, ok := styleStyleFuncs[key]; ok {
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

////////  Style

// styleStyleFuncs are functions for styling the Style object itself
var styleStyleFuncs = map[string]styleprops.Func{
	"color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.Color = parent.(*Style).Color
			} else if init {
				fs.Color = colors.Scheme.OnSurface
			}
			return
		}
		fs.Color = errors.Log1(gradient.FromAny(val, cc))
	},
	"background-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.Background = parent.(*Style).Background
			} else if init {
				fs.Background = nil
			}
			return
		}
		fs.Background = errors.Log1(gradient.FromAny(val, cc))
	},
	"opacity": styleprops.Float(float32(1),
		func(obj *Style) *float32 { return &obj.Opacity }),
}

////////  Layout

// styleLayoutFuncs are functions for styling the layout
// style properties; they are still stored on the main style object,
// but they are done separately to improve clarity
var styleLayoutFuncs = map[string]styleprops.Func{
	"display": styleprops.Enum(Flex,
		func(obj *Style) enums.EnumSetter { return &obj.Display }),
	"flex-direction": func(obj any, key string, val, parent any, cc colors.Context) {
		s := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				s.Direction = parent.(*Style).Direction
			} else if init {
				s.Direction = Row
			}
			return
		}
		str := reflectx.ToString(val)
		if str == "row" || str == "row-reverse" {
			s.Direction = Row
		} else {
			s.Direction = Column
		}
	},
	// TODO(kai/styproperties): multi-dim flex-grow
	"flex-grow": styleprops.Float(0, func(obj *Style) *float32 { return &obj.Grow.Y }),
	"wrap": styleprops.Bool(false,
		func(obj *Style) *bool { return &obj.Wrap }),
	"justify-content": styleprops.Enum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Justify.Content }),
	"justify-items": styleprops.Enum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Justify.Items }),
	"justify-self": styleprops.Enum(Auto,
		func(obj *Style) enums.EnumSetter { return &obj.Justify.Self }),
	"align-content": styleprops.Enum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Align.Content }),
	"align-items": styleprops.Enum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Align.Items }),
	"align-self": styleprops.Enum(Auto,
		func(obj *Style) enums.EnumSetter { return &obj.Align.Self }),
	"x": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Pos.X }),
	"y": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Pos.Y }),
	"width": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Min.X }),
	"height": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Min.Y }),
	"max-width": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Max.X }),
	"max-height": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Max.Y }),
	"min-width": styleprops.Units(units.Dp(2),
		func(obj *Style) *units.Value { return &obj.Min.X }),
	"min-height": styleprops.Units(units.Dp(2),
		func(obj *Style) *units.Value { return &obj.Min.Y }),
	"margin": func(obj any, key string, val any, parent any, cc colors.Context) {
		s := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
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
		if inh, init := styleprops.InhInit(val, parent); inh || init {
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
	"overflow": styleprops.Enum(OverflowAuto,
		func(obj *Style) enums.EnumSetter { return &obj.Overflow.Y }),
	"columns": styleprops.Int(int(0),
		func(obj *Style) *int { return &obj.Columns }),
	"row": styleprops.Int(int(0),
		func(obj *Style) *int { return &obj.Row }),
	"col": styleprops.Int(int(0),
		func(obj *Style) *int { return &obj.Col }),
	"row-span": styleprops.Int(int(0),
		func(obj *Style) *int { return &obj.RowSpan }),
	"col-span": styleprops.Int(int(0),
		func(obj *Style) *int { return &obj.ColSpan }),
	"z-index": styleprops.Int(int(0),
		func(obj *Style) *int { return &obj.ZIndex }),
	"scrollbar-width": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.ScrollbarWidth }),
}

////////  Font

// styleFontFuncs are functions for styling the Font object
var styleFontFuncs = map[string]styleprops.Func{
	"font-size": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Font)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
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
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.Family = parent.(*Font).Family
			} else if init {
				fs.Family = "" // font has defaults
			}
			return
		}
		fs.Family = reflectx.ToString(val)
	},
	"font-style": styleprops.Enum(FontNormal,
		func(obj *Font) enums.EnumSetter { return &obj.Style }),
	"font-weight": styleprops.Enum(WeightNormal,
		func(obj *Font) enums.EnumSetter { return &obj.Weight }),
	"font-stretch": styleprops.Enum(FontStrNormal,
		func(obj *Font) enums.EnumSetter { return &obj.Stretch }),
	"font-variant": styleprops.Enum(FontVarNormal,
		func(obj *Font) enums.EnumSetter { return &obj.Variant }),
	"baseline-shift": styleprops.Enum(ShiftBaseline,
		func(obj *Font) enums.EnumSetter { return &obj.Shift }),
	"text-decoration": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Font)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
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
			iv, err := reflectx.ToInt(val)
			if err == nil {
				fs.Decoration = TextDecorations(iv)
			} else {
				styleprops.SetError(key, val, err)
			}
		}
	},
}

// styleFontRenderFuncs are _extra_ functions for styling
// the FontRender object in addition to base Font
var styleFontRenderFuncs = map[string]styleprops.Func{
	"color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*FontRender)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.Color = parent.(*FontRender).Color
			} else if init {
				fs.Color = colors.Scheme.OnSurface
			}
			return
		}
		fs.Color = errors.Log1(gradient.FromAny(val, cc))
	},
	"background-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*FontRender)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.Background = parent.(*FontRender).Background
			} else if init {
				fs.Background = nil
			}
			return
		}
		fs.Background = errors.Log1(gradient.FromAny(val, cc))
	},
	"opacity": styleprops.Float(float32(1),
		func(obj *FontRender) *float32 { return &obj.Opacity }),
}

/////////////////////////////////////////////////////////////////////////////////
//  Text

// styleTextFuncs are functions for styling the Text object
var styleTextFuncs = map[string]styleprops.Func{
	"text-align": styleprops.Enum(Start,
		func(obj *Text) enums.EnumSetter { return &obj.Align }),
	"text-vertical-align": styleprops.Enum(Start,
		func(obj *Text) enums.EnumSetter { return &obj.AlignV }),
	"text-anchor": styleprops.Enum(AnchorStart,
		func(obj *Text) enums.EnumSetter { return &obj.Anchor }),
	"letter-spacing": styleprops.Units(units.Value{},
		func(obj *Text) *units.Value { return &obj.LetterSpacing }),
	"word-spacing": styleprops.Units(units.Value{},
		func(obj *Text) *units.Value { return &obj.WordSpacing }),
	"line-height": styleprops.Units(LineHeightNormal,
		func(obj *Text) *units.Value { return &obj.LineHeight }),
	"white-space": styleprops.Enum(WhiteSpaceNormal,
		func(obj *Text) enums.EnumSetter { return &obj.WhiteSpace }),
	"unicode-bidi": styleprops.Enum(BidiNormal,
		func(obj *Text) enums.EnumSetter { return &obj.UnicodeBidi }),
	"direction": styleprops.Enum(LRTB,
		func(obj *Text) enums.EnumSetter { return &obj.Direction }),
	"writing-mode": styleprops.Enum(LRTB,
		func(obj *Text) enums.EnumSetter { return &obj.WritingMode }),
	"glyph-orientation-vertical": styleprops.Float(float32(1),
		func(obj *Text) *float32 { return &obj.OrientationVert }),
	"glyph-orientation-horizontal": styleprops.Float(float32(1),
		func(obj *Text) *float32 { return &obj.OrientationHoriz }),
	"text-indent": styleprops.Units(units.Value{},
		func(obj *Text) *units.Value { return &obj.Indent }),
	"para-spacing": styleprops.Units(units.Value{},
		func(obj *Text) *units.Value { return &obj.ParaSpacing }),
	"tab-size": styleprops.Int(int(4),
		func(obj *Text) *int { return &obj.TabSize }),
}

/////////////////////////////////////////////////////////////////////////////////
//  Border

// styleBorderFuncs are functions for styling the Border object
var styleBorderFuncs = map[string]styleprops.Func{
	// SidesTODO: need to figure out how to get key and context information for side SetAny calls
	// with padding, margin, border, etc
	"border-style": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
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
			iv, err := reflectx.ToInt(val)
			if err == nil {
				bs.Style.Set(BorderStyles(iv))
			} else {
				styleprops.SetError(key, val, err)
			}
		}
	},
	"border-width": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
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
		if inh, init := styleprops.InhInit(val, parent); inh || init {
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
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				bs.Color = parent.(*Border).Color
			} else if init {
				bs.Color.Set(colors.Scheme.Outline)
			}
			return
		}
		// TODO(kai): support side-specific border colors
		bs.Color.Set(errors.Log1(gradient.FromAny(val, cc)))
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Outline

// styleOutlineFuncs are functions for styling the OutlineStyle object
var styleOutlineFuncs = map[string]styleprops.Func{
	"outline-style": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
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
			iv, err := reflectx.ToInt(val)
			if err == nil {
				bs.Style.Set(BorderStyles(iv))
			} else {
				styleprops.SetError(key, val, err)
			}
		}
	},
	"outline-width": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
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
		if inh, init := styleprops.InhInit(val, parent); inh || init {
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
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				bs.Color = parent.(*Border).Color
			} else if init {
				bs.Color.Set(colors.Scheme.Outline)
			}
			return
		}
		// TODO(kai): support side-specific border colors
		bs.Color.Set(errors.Log1(gradient.FromAny(val, cc)))
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Shadow

// styleShadowFuncs are functions for styling the Shadow object
var styleShadowFuncs = map[string]styleprops.Func{
	"box-shadow.offset-x": styleprops.Units(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.OffsetX }),
	"box-shadow.offset-y": styleprops.Units(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.OffsetY }),
	"box-shadow.blur": styleprops.Units(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.Blur }),
	"box-shadow.spread": styleprops.Units(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.Spread }),
	"box-shadow.color": func(obj any, key string, val any, parent any, cc colors.Context) {
		ss := obj.(*Shadow)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				ss.Color = parent.(*Shadow).Color
			} else if init {
				ss.Color = colors.Scheme.Shadow
			}
			return
		}
		ss.Color = errors.Log1(gradient.FromAny(val, cc))
	},
	"box-shadow.inset": styleprops.Bool(false,
		func(obj *Shadow) *bool { return &obj.Inset }),
}
