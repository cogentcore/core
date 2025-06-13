// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/styles/styleprops"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
)

// FromProperties sets style field values based on the given property list.
func (s *Style) FromProperties(parent *Style, properties map[string]any, ctxt colors.Context) {
	for key, val := range properties {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		s.FromProperty(parent, key, val, ctxt)
	}
}

// FromProperty sets style field values based on the given property key and value.
func (s *Style) FromProperty(parent *Style, key string, val any, cc colors.Context) {
	if sfunc, ok := styleFuncs[key]; ok {
		if parent != nil {
			sfunc(s, key, val, parent, cc)
		} else {
			sfunc(s, key, val, nil, cc)
		}
		return
	}
}

// styleFuncs are functions for styling the text.Style object.
var styleFuncs = map[string]styleprops.Func{
	"text-align": styleprops.Enum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Align }),
	"text-vertical-align": styleprops.Enum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.AlignV }),
	// note: text-style reads the font-size setting for regular units cases.
	"font-size": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.FontSize }),
	"line-height": styleprops.FloatProportion(float32(1.3),
		func(obj *Style) *float32 { return &obj.LineHeight }),
	"line-spacing": styleprops.FloatProportion(float32(1.3),
		func(obj *Style) *float32 { return &obj.LineHeight }),
	"para-spacing": styleprops.FloatProportion(float32(1.3),
		func(obj *Style) *float32 { return &obj.ParaSpacing }),
	"white-space": styleprops.Enum(WrapAsNeeded,
		func(obj *Style) enums.EnumSetter { return &obj.WhiteSpace }),
	"direction": styleprops.Enum(rich.LTR,
		func(obj *Style) enums.EnumSetter { return &obj.Direction }),
	"text-indent": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Indent }),
	"tab-size": styleprops.Int(int(4),
		func(obj *Style) *int { return &obj.TabSize }),
	"select-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.SelectColor = parent.(*Style).SelectColor
			} else if init {
				fs.SelectColor = colors.Scheme.Select.Container
			}
			return
		}
		fs.SelectColor = errors.Log1(gradient.FromAny(val, cc))
	},
	"highlight-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.HighlightColor = parent.(*Style).HighlightColor
			} else if init {
				fs.HighlightColor = colors.Scheme.Warn.Container
			}
			return
		}
		fs.HighlightColor = errors.Log1(gradient.FromAny(val, cc))
	},
}

// ToProperties sets map[string]any properties based on non-default style values.
// properties map must be non-nil.
func (s *Style) ToProperties(sty *rich.Style, p map[string]any) {
	if s.FontSize.Unit != units.UnitDp || s.FontSize.Value != 16 || sty.Size != 1 {
		sz := s.FontSize
		sz.Value *= sty.Size
		p["font-size"] = sz.StringCSS()
	}
	if sty.Family == rich.Custom && s.CustomFont != "" {
		p["font-family"] = s.CustomFont
	}
	if sty.Slant != rich.SlantNormal {
		p["font-style"] = sty.Slant.String()
	}
	if sty.Weight != rich.Normal {
		p["font-weight"] = sty.Weight.String()
	}
	if sty.Stretch != rich.StretchNormal {
		p["font-stretch"] = sty.Stretch.String()
	}
	if dstr := sty.Decoration.String(); dstr != "" {
		p["text-decoration"] = dstr
	} else {
		p["text-decoration"] = "none"
	}
	if s.Align != Start {
		p["text-align"] = s.Align.String()
	}
	if s.AlignV != Start {
		p["text-vertical-align"] = s.AlignV.String()
	}
	if s.LineHeight != 1.3 {
		p["line-height"] = reflectx.ToString(s.LineHeight)
	}
	if s.WhiteSpace != WrapAsNeeded {
		p["white-space"] = s.WhiteSpace.String()
	}
	if sty.Direction != rich.LTR {
		p["direction"] = s.Direction.String()
	}
	if s.TabSize != 4 {
		p["tab-size"] = reflectx.ToString(s.TabSize)
	}
	p["fill"] = colors.AsHex(s.FillColor(sty))
	if sc := sty.StrokeColor(); sc != nil {
		p["stroke-color"] = colors.AsHex(sc)
	}
	if s.SelectColor != nil {
		p["select-color"] = colors.AsHex(colors.ToUniform(s.SelectColor))
	}
	if s.HighlightColor != nil {
		p["highlight-color"] = colors.AsHex(colors.ToUniform(s.HighlightColor))
	}
}
