// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"log"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles/styleprops"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

/////// see style_properties.go for master version

// styleFromProperties sets style field values based on map[string]any properties
func (pc *Path) styleFromProperties(parent *Path, properties map[string]any, cc colors.Context) {
	for key, val := range properties {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		if key == "display" {
			if inh, init := styleprops.InhInit(val, parent); inh || init {
				if inh {
					pc.Display = parent.Display
				} else if init {
					pc.Display = true
				}
				return
			}
			sval := reflectx.ToString(val)
			switch sval {
			case "none":
				pc.Display = false
			case "inline":
				pc.Display = true
			default:
				pc.Display = true
			}
			continue
		}
		if sfunc, ok := styleStrokeFuncs[key]; ok {
			if parent != nil {
				sfunc(&pc.Stroke, key, val, &parent.Stroke, cc)
			} else {
				sfunc(&pc.Stroke, key, val, nil, cc)
			}
			continue
		}
		if sfunc, ok := styleFillFuncs[key]; ok {
			if parent != nil {
				sfunc(&pc.Fill, key, val, &parent.Fill, cc)
			} else {
				sfunc(&pc.Fill, key, val, nil, cc)
			}
			continue
		}
		if sfunc, ok := stylePathFuncs[key]; ok {
			sfunc(pc, key, val, parent, cc)
			continue
		}
	}
}

// styleFromProperties sets style field values based on map[string]any properties
func (pc *Paint) styleFromProperties(parent *Paint, properties map[string]any, cc colors.Context) {
	var ppath *Path
	var pfont *rich.Style
	var ptext *text.Style
	if parent != nil {
		ppath = &parent.Path
		pfont = &parent.Font
		ptext = &parent.Text
	}
	pc.Path.styleFromProperties(ppath, properties, cc)
	pc.Font.StyleFromProperties(pfont, properties, cc)
	pc.Text.StyleFromProperties(ptext, properties, cc)
	for key, val := range properties {
		_ = val
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		// todo: add others here
	}
}

////////  Stroke

// styleStrokeFuncs are functions for styling the Stroke object
var styleStrokeFuncs = map[string]styleprops.Func{
	"stroke": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Stroke)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.Color = parent.(*Stroke).Color
			} else if init {
				fs.Color = colors.Uniform(colors.Black)
			}
			return
		}
		fs.Color = errors.Log1(gradient.FromAny(val, cc))
	},
	"stroke-opacity": styleprops.Float(float32(1),
		func(obj *Stroke) *float32 { return &(obj.Opacity) }),
	"opacity": styleprops.Float(float32(1),
		func(obj *Stroke) *float32 { return &(obj.Opacity) }),
	"stroke-width": styleprops.Units(units.Dp(1),
		func(obj *Stroke) *units.Value { return &(obj.Width) }),
	"stroke-min-width": styleprops.Units(units.Dp(1),
		func(obj *Stroke) *units.Value { return &(obj.MinWidth) }),
	"stroke-dasharray": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Stroke)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.Dashes = parent.(*Stroke).Dashes
			} else if init {
				fs.Dashes = nil
			}
			return
		}
		switch vt := val.(type) {
		case string:
			fs.Dashes = parseDashesString(vt)
		case []float32:
			math32.CopyFloat32s(&fs.Dashes, vt)
		case *[]float32:
			math32.CopyFloat32s(&fs.Dashes, *vt)
		}
	},
	"stroke-linecap": styleprops.Enum(ppath.CapButt,
		func(obj *Stroke) enums.EnumSetter { return &(obj.Cap) }),
	"stroke-linejoin": styleprops.Enum(ppath.JoinMiter,
		func(obj *Stroke) enums.EnumSetter { return &(obj.Join) }),
	"stroke-miterlimit": styleprops.Float(float32(1),
		func(obj *Stroke) *float32 { return &(obj.MiterLimit) }),
}

////////  Fill

// styleFillFuncs are functions for styling the Fill object
var styleFillFuncs = map[string]styleprops.Func{
	"fill": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Fill)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.Color = parent.(*Fill).Color
			} else if init {
				fs.Color = colors.Uniform(colors.Black)
			}
			return
		}
		fs.Color = errors.Log1(gradient.FromAny(val, cc))
	},
	"fill-opacity": styleprops.Float(float32(1),
		func(obj *Fill) *float32 { return &(obj.Opacity) }),
	"opacity": styleprops.Float(float32(1),
		func(obj *Fill) *float32 { return &(obj.Opacity) }),
	"fill-rule": styleprops.Enum(ppath.NonZero,
		func(obj *Fill) enums.EnumSetter { return &(obj.Rule) }),
}

////////  Paint

// stylePathFuncs are functions for styling the Stroke object
var stylePathFuncs = map[string]styleprops.Func{
	"vector-effect": styleprops.Enum(ppath.VectorEffectNone,
		func(obj *Path) enums.EnumSetter { return &(obj.VectorEffect) }),
	"transform": func(obj any, key string, val any, parent any, cc colors.Context) {
		pc := obj.(*Path)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				pc.Transform = parent.(*Path).Transform
			} else if init {
				pc.Transform = math32.Identity2()
			}
			return
		}
		switch vt := val.(type) {
		case string:
			pc.Transform.SetString(vt)
		case *math32.Matrix2:
			pc.Transform = *vt
		case math32.Matrix2:
			pc.Transform = vt
		}
	},
}

// parseDashesString gets a dash slice from given string
func parseDashesString(str string) []float32 {
	if len(str) == 0 || str == "none" {
		return nil
	}
	ds := strings.Split(str, ",")
	dl := make([]float32, len(ds))
	for i, dstr := range ds {
		d, err := strconv.ParseFloat(strings.TrimSpace(dstr), 32)
		if err != nil {
			log.Printf("core.ParseDashesString parsing error: %v\n", err)
			return nil
		}
		dl[i] = float32(d)
	}
	return dl
}
