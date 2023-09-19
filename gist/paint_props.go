// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"log"
	"strconv"
	"strings"

	"goki.dev/colors"
	"goki.dev/enums"
	"goki.dev/girl/units"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
)

////////////////////////////////////////////////////////////////////////////
//   Styling functions for setting from properties
//     see style_props.go for master version

// StyleFromProps sets style field values based on map[string]any properties
func (pc *Paint) StyleFromProps(par *Paint, props map[string]any, ctxt Context) {
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
	"stroke": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Stroke)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*Stroke).Color
			} else if init {
				fs.SetColor(colors.Black)
			}
			return
		}
		fs.Color.SetIFace(val, ctxt, key)
	},
	"stroke-opacity": StyleFuncFloat(float32(1),
		func(obj *Stroke) *float32 { return &(obj.Opacity) }),
	"stroke-width": StyleFuncUnits(units.Px(1),
		func(obj *Stroke) *units.Value { return &(obj.Width) }),
	"stroke-min-width": StyleFuncUnits(units.Px(1),
		func(obj *Stroke) *units.Value { return &(obj.MinWidth) }),
	"stroke-dasharray": func(obj any, key string, val any, par any, ctxt Context) {
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
		case []float64:
			mat32.CopyFloat64s(&fs.Dashes, vt)
		case *[]float64:
			mat32.CopyFloat64s(&fs.Dashes, *vt)
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
	"fill": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*Fill)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*Fill).Color
			} else if init {
				fs.SetColor(colors.Black)
			}
			return
		}
		fs.Color.SetIFace(val, ctxt, key)
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
	"transform": func(obj any, key string, val any, par any, ctxt Context) {
		pc := obj.(*Paint)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				pc.XForm = par.(*Paint).XForm
			} else if init {
				pc.XForm = mat32.Identity2D()
			}
			return
		}
		switch vt := val.(type) {
		case string:
			pc.XForm.SetString(vt)
		case *mat32.Mat2:
			pc.XForm = *vt
		case mat32.Mat2:
			pc.XForm = vt
		}
	},
}

// ParseDashesString gets a dash slice from given string
func ParseDashesString(str string) []float64 {
	if len(str) == 0 || str == "none" {
		return nil
	}
	ds := strings.Split(str, ",")
	dl := make([]float64, len(ds))
	for i, dstr := range ds {
		d, err := strconv.ParseFloat(strings.TrimSpace(dstr), 64)
		if err != nil {
			log.Printf("gi.ParseDashesString parsing error: %v\n", err)
			return nil
		}
		dl[i] = d
	}
	return dl
}

// StyleFontRenderFuncs are functions for styling the FontStyle fields beyond Font
var StyleFontRenderFuncs = map[string]StyleFunc{
	"color": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*FontRender)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*FontRender).Color
			} else if init {
				fs.Color = colors.Black
			}
			return
		}
		fs.Color = colors.LogFromAny(val, ctxt.ContextColor())
	},
	"background-color": func(obj any, key string, val any, par any, ctxt Context) {
		fs := obj.(*FontRender)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.BackgroundColor = par.(*FontRender).BackgroundColor
			} else if init {
				fs.BackgroundColor = ColorSpec{}
			}
			return
		}
		fs.BackgroundColor.SetIFace(val, ctxt, key)
	},
}
