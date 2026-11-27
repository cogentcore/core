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
func (ts *Style) FromProperties(parent *Style, properties map[string]any, ctxt colors.Context) {
	for key, val := range properties {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		ts.FromProperty(parent, key, val, ctxt)
	}
}

// FromProperty sets style field values based on the given property key and value.
func (ts *Style) FromProperty(parent *Style, key string, val any, cc colors.Context) {
	if sfunc, ok := styleFuncs[key]; ok {
		if parent != nil {
			sfunc(ts, key, val, parent, cc)
		} else {
			sfunc(ts, key, val, nil, cc)
		}
		return
	}
}

// styleFuncs are functions for styling the text.Style object.
var styleFuncs = map[string]styleprops.Func{
	"text-align": AlignProperty(func(obj *Style) enums.EnumSetter { return &obj.Align }),
	"text-vertical-align": styleprops.Enum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.AlignV }),
	// note: text-style reads the font-size setting for regular units cases.
	"font-size": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.FontSize }),
	"line-height": styleprops.FloatProportion(float32(1.3),
		func(obj *Style) *float32 { return &obj.LineHeight }),
	"line-spacing": styleprops.FloatProportion(float32(1.3),
		func(obj *Style) *float32 { return &obj.LineHeight }),
	"para-spacing": styleprops.FloatProportion(float32(1.5),
		func(obj *Style) *float32 { return &obj.ParaSpacing }),
	"white-space": styleprops.Enum(WrapAsNeeded,
		func(obj *Style) enums.EnumSetter { return &obj.WhiteSpace }),
	"direction": styleprops.Enum(rich.LTR,
		func(obj *Style) enums.EnumSetter { return &obj.Direction }),
	"text-indent": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Indent }),
	"text-hanging": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Hanging }),
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
func (ts *Style) ToProperties(sty *rich.Style, p map[string]any) {
	if ts.FontSize.Unit != units.UnitDp || ts.FontSize.Value != 16 || sty.Size != 1 {
		sz := ts.FontSize
		sz.Value *= sty.Size
		p["font-size"] = sz.StringCSS()
	}
	if sty.Family == rich.Custom && ts.CustomFont != "" {
		p["font-family"] = ts.CustomFont
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
	if ts.Align != Start {
		p["text-align"] = ts.Align.String()
	}
	if ts.AlignV != Start {
		p["text-vertical-align"] = ts.AlignV.String()
	}
	if ts.LineHeight != 1.3 {
		p["line-height"] = reflectx.ToString(ts.LineHeight)
	}
	if ts.WhiteSpace != WrapAsNeeded {
		p["white-space"] = ts.WhiteSpace.String()
	}
	if ts.Direction != rich.LTR {
		p["direction"] = ts.Direction.String()
	}
	if ts.TabSize != 4 {
		p["tab-size"] = reflectx.ToString(ts.TabSize)
	}
	p["fill"] = colors.AsHex(ts.FillColor(sty))
	if sc := sty.StrokeColor(); sc != nil {
		p["stroke-color"] = colors.AsHex(sc)
	}
	if ts.SelectColor != nil {
		p["select-color"] = colors.AsHex(colors.ToUniform(ts.SelectColor))
	}
	if ts.HighlightColor != nil {
		p["highlight-color"] = colors.AsHex(colors.ToUniform(ts.HighlightColor))
	}
}

// AlignProperty returns the style function of the text-align property of
// the given field of the alignment of text, of type [Aligns], whose initial
// value is [Start], and for which the CSS values left and right are
// [Start] and [End], as in the default left-to-right direction.
func AlignProperty[T any](getField func(obj *T) enums.EnumSetter) styleprops.Func {
	f := styleprops.Enum(Start, getField)
	return func(obj any, key string, val any, parent any, cc colors.Context) {
		if s, ok := val.(string); ok {
			switch s {
			case "left":
				val = "start"
			case "right":
				val = "end"
			}
		}
		f(obj, key, val, parent, cc)
	}
}
