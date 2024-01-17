// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"log"
	"strconv"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/units"
)

////////////////////////////////////////////////////////////////////////////
//   Styling functions for setting from properties
//     see style_props.go for master version

// StyleFromProps sets style field values based on map[string]any properties
func (pc *Paint) StyleFromProps(par *Paint, props map[string]any, ctxt colors.Context) {
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		if key == "display" {
			if inh, init := StyleInhInit(val, par); inh || init {
				if inh {
					pc.Display = par.Display
				} else if init {
					pc.Display = true
				}
				return
			}
			sval := laser.ToString(val)
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
		if sfunc, ok := StyleStrokeFuncs[key]; ok {
			if par != nil {
				sfunc(&pc.StrokeStyle, key, val, &par.StrokeStyle, ctxt)
			} else {
				sfunc(&pc.StrokeStyle, key, val, nil, ctxt)
			}
			continue
		}
		if sfunc, ok := StyleFillFuncs[key]; ok {
			if par != nil {
				sfunc(&pc.FillStyle, key, val, &par.FillStyle, ctxt)
			} else {
				sfunc(&pc.FillStyle, key, val, nil, ctxt)
			}
			continue
		}
		if sfunc, ok := StyleFontFuncs[key]; ok {
			if par != nil {
				sfunc(&pc.FontStyle.Font, key, val, &par.FontStyle.Font, ctxt)
			} else {
				sfunc(&pc.FontStyle.Font, key, val, nil, ctxt)
			}
			continue
		}
		if sfunc, ok := StyleFontRenderFuncs[key]; ok {
			if par != nil {
				sfunc(&pc.FontStyle, key, val, &par.FontStyle, ctxt)
			} else {
				sfunc(&pc.FontStyle, key, val, nil, ctxt)
			}
			continue
		}
		if sfunc, ok := StyleTextFuncs[key]; ok {
			if par != nil {
				sfunc(&pc.TextStyle, key, val, &par.TextStyle, ctxt)
			} else {
				sfunc(&pc.TextStyle, key, val, nil, ctxt)
			}
			continue
		}
		if sfunc, ok := StylePaintFuncs[key]; ok {
			sfunc(pc, key, val, par, ctxt)
			continue
		}
	}
}

/////////////////////////////////////////////////////////////////////////////////
//  Stroke

// StyleStrokeFuncs are functions for styling the Stroke object
var StyleStrokeFuncs = map[string]StyleFunc{
	"stroke": func(obj any, key string, val any, par any, ctxt colors.Context) {
		fs := obj.(*Stroke)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*Stroke).Color
			} else if init {
				fs.Color = colors.C(colors.Black)
			}
			return
		}
		fs.Color = grr.Log1(gradient.FromAny(val, ctxt))
	},
	"stroke-opacity": StyleFuncFloat(float32(1),
		func(obj *Stroke) *float32 { return &(obj.Opacity) }),
	"stroke-width": StyleFuncUnits(units.Dp(1),
		func(obj *Stroke) *units.Value { return &(obj.Width) }),
	"stroke-min-width": StyleFuncUnits(units.Dp(1),
		func(obj *Stroke) *units.Value { return &(obj.MinWidth) }),
	"stroke-dasharray": func(obj any, key string, val any, par any, ctxt colors.Context) {
		fs := obj.(*Stroke)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Dashes = par.(*Stroke).Dashes
			} else if init {
				fs.Dashes = nil
			}
			return
		}
		switch vt := val.(type) {
		case string:
			fs.Dashes = ParseDashesString(vt)
		case []float32:
			mat32.CopyFloat32s(&fs.Dashes, vt)
		case *[]float32:
			mat32.CopyFloat32s(&fs.Dashes, *vt)
		}
	},
	"stroke-linecap": StyleFuncEnum(LineCapButt,
		func(obj *Stroke) enums.EnumSetter { return &(obj.Cap) }),
	"stroke-linejoin": StyleFuncEnum(LineJoinMiter,
		func(obj *Stroke) enums.EnumSetter { return &(obj.Join) }),
	"stroke-miterlimit": StyleFuncFloat(float32(1),
		func(obj *Stroke) *float32 { return &(obj.MiterLimit) }),
}

/////////////////////////////////////////////////////////////////////////////////
//  Fill

// StyleFillFuncs are functions for styling the Fill object
var StyleFillFuncs = map[string]StyleFunc{
	"fill": func(obj any, key string, val any, par any, ctxt colors.Context) {
		fs := obj.(*Fill)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*Fill).Color
			} else if init {
				fs.Color = colors.C(colors.Black)
			}
			return
		}
		fs.Color = grr.Log1(gradient.FromAny(val, ctxt))
	},
	"fill-opacity": StyleFuncFloat(float32(1),
		func(obj *Fill) *float32 { return &(obj.Opacity) }),
	"fill-rule": StyleFuncEnum(FillRuleNonZero,
		func(obj *Fill) enums.EnumSetter { return &(obj.Rule) }),
}

/////////////////////////////////////////////////////////////////////////////////
//  Paint

// StylePaintFuncs are functions for styling the Stroke object
var StylePaintFuncs = map[string]StyleFunc{
	"vector-effect": StyleFuncEnum(VecEffNone,
		func(obj *Paint) enums.EnumSetter { return &(obj.VecEff) }),
	"transform": func(obj any, key string, val any, par any, ctxt colors.Context) {
		pc := obj.(*Paint)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				pc.Transform = par.(*Paint).Transform
			} else if init {
				pc.Transform = mat32.Identity2()
			}
			return
		}
		switch vt := val.(type) {
		case string:
			pc.Transform.SetString(vt)
		case *mat32.Mat2:
			pc.Transform = *vt
		case mat32.Mat2:
			pc.Transform = vt
		}
	},
}

// ParseDashesString gets a dash slice from given string
func ParseDashesString(str string) []float32 {
	if len(str) == 0 || str == "none" {
		return nil
	}
	ds := strings.Split(str, ",")
	dl := make([]float32, len(ds))
	for i, dstr := range ds {
		d, err := strconv.ParseFloat(strings.TrimSpace(dstr), 32)
		if err != nil {
			log.Printf("gi.ParseDashesString parsing error: %v\n", err)
			return nil
		}
		dl[i] = float32(d)
	}
	return dl
}
