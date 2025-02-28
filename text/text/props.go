// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/styles/styleprops"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
)

// StyleFromProperties sets style field values based on the given property list.
func (s *Style) StyleFromProperties(parent *Style, properties map[string]any, ctxt colors.Context) {
	for key, val := range properties {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		s.StyleFromProperty(parent, key, val, ctxt)
	}
}

// StyleFromProperty sets style field values based on the given property key and value.
func (s *Style) StyleFromProperty(parent *Style, key string, val any, cc colors.Context) {
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
	"line-height": styleprops.Float(float32(1.2),
		func(obj *Style) *float32 { return &obj.LineSpacing }),
	"line-spacing": styleprops.Float(float32(1.2),
		func(obj *Style) *float32 { return &obj.LineSpacing }),
	"para-spacing": styleprops.Float(float32(1.2),
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
				fs.SelectColor = colors.Uniform(colors.Black)
			}
			return
		}
		fs.SelectColor = errors.Log1(gradient.FromAny(val, cc))
	},
}
