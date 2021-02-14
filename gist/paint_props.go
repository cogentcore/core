// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"log"
	"strconv"
	"strings"

	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

////////////////////////////////////////////////////////////////////////////
//   Styling functions for setting from properties
//     see style_props.go for master version

// StyleFromProps sets style field values based on ki.Props properties
func (pc *Paint) StyleFromProps(par *Paint, props ki.Props, ctxt Context) {
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
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
	"stroke": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Stroke)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*Stroke).Color
			} else if init {
				fs.SetColor(Black)
			}
			return
		}
		fs.Color.SetIFace(val, ctxt, key)
	},
	"stroke-opacity": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Stroke)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Opacity = par.(*Stroke).Opacity
			} else if init {
				fs.Opacity = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			fs.Opacity = iv
		}
	},
	"stroke-width": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Stroke)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Width = par.(*Stroke).Width
			} else if init {
				fs.Width.Set(1, units.Px)
			}
			return
		}
		fs.Width.SetIFace(val, key)
	},
	"stroke-min-width": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Stroke)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.MinWidth = par.(*Stroke).MinWidth
			} else if init {
				fs.MinWidth.Set(1, units.Px)
			}
			return
		}
		fs.MinWidth.SetIFace(val, key)
	},
	"stroke-dasharray": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
	"stroke-linecap": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Stroke)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Cap = par.(*Stroke).Cap
			} else if init {
				fs.Cap = LineCapButt
			}
			return
		}
		switch vt := val.(type) {
		case string:
			fs.Cap.FromString(vt)
		case LineCaps:
			fs.Cap = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				fs.Cap = LineCaps(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"stroke-linejoin": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Stroke)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Join = par.(*Stroke).Join
			} else if init {
				fs.Join = LineJoinMiter
			}
			return
		}
		switch vt := val.(type) {
		case string:
			fs.Join.FromString(vt)
		case LineJoins:
			fs.Join = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				fs.Join = LineJoins(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"stroke-miterlimit": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Stroke)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.MiterLimit = par.(*Stroke).MiterLimit
			} else if init {
				fs.MiterLimit = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			fs.MiterLimit = iv
		}
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Fill

// StyleFillFuncs are functions for styling the Fill object
var StyleFillFuncs = map[string]StyleFunc{
	"fill": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Fill)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*Fill).Color
			} else if init {
				fs.SetColor(Black)
			}
			return
		}
		fs.Color.SetIFace(val, ctxt, key)
	},
	"fill-opacity": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Fill)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Opacity = par.(*Fill).Opacity
			} else if init {
				fs.Opacity = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			fs.Opacity = iv
		}
	},
	"fill-rule": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		fs := obj.(*Fill)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Rule = par.(*Fill).Rule
			} else if init {
				fs.Rule = FillRuleNonZero
			}
			return
		}
		switch vt := val.(type) {
		case string:
			fs.Rule.FromString(vt)
		case FillRules:
			fs.Rule = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				fs.Rule = FillRules(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Paint

// StylePaintFuncs are functions for styling the Stroke object
var StylePaintFuncs = map[string]StyleFunc{
	"vector-effect": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
		pc := obj.(*Paint)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				pc.VecEff = par.(*Paint).VecEff
			} else if init {
				pc.VecEff = VecEffNone
			}
			return
		}
		switch vt := val.(type) {
		case string:
			pc.VecEff.FromString(vt)
		case VectorEffects:
			pc.VecEff = vt
		default:
			if iv, ok := kit.ToInt(val); ok {
				pc.VecEff = VectorEffects(iv)
			} else {
				StyleSetError(key, val)
			}
		}
	},
	"transform": func(obj interface{}, key string, val interface{}, par interface{}, ctxt Context) {
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
