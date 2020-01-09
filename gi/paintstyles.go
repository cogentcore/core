// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image/color"
	"log"
	"strconv"
	"strings"

	"github.com/goki/gi/mat32"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

type FillRules int

const (
	FillRuleNonZero FillRules = iota
	FillRuleEvenOdd
	FillRulesN
)

//go:generate stringer -type=FillRules

var KiT_FillRules = kit.Enums.AddEnumAltLower(FillRulesN, kit.NotBitFlag, StylePropProps, "FillRules")

func (ev FillRules) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *FillRules) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// IMPORTANT: any changes here must be updated below in StyleFillFuncs

// FillStyle contains all the properties for filling a region
type FillStyle struct {
	On      bool      `desc:"is fill active -- if property is none then false"`
	Color   ColorSpec `xml:"fill" desc:"prop: fill = fill color specification"`
	Opacity float32   `xml:"fill-opacity" desc:"prop: fill-opacity = global alpha opacity / transparency factor"`
	Rule    FillRules `xml:"fill-rule" desc:"prop: fill-rule = rule for how to fill more complex shapes with crossing lines"`
}

// Defaults initializes default values for paint fill
func (pf *FillStyle) Defaults() {
	pf.On = true // svg says fill is ON by default
	pf.SetColor(color.Black)
	pf.Rule = FillRuleNonZero
	pf.Opacity = 1.0
}

// SetStylePost does some updating after setting the style from user properties
func (pf *FillStyle) SetStylePost(props ki.Props) {
	if pf.Color.IsNil() {
		pf.On = false
	} else {
		pf.On = true
	}
}

// SetColor sets a solid fill color -- nil turns off filling
func (pf *FillStyle) SetColor(cl color.Color) {
	if cl == nil {
		pf.On = false
	} else {
		pf.On = true
		pf.Color.Color.SetColor(cl)
		pf.Color.Source = SolidColor
	}
}

// SetColorSpec sets full color spec from source
func (pf *FillStyle) SetColorSpec(cl *ColorSpec) {
	if cl == nil {
		pf.On = false
	} else {
		pf.On = true
		pf.Color.CopyFrom(cl)
	}
}

////////////////////////////////////////////////////////////////////////////////////
// Stroke

// end-cap of a line: stroke-linecap property in SVG
type LineCaps int

const (
	LineCapButt LineCaps = iota
	LineCapRound
	LineCapSquare
	// rasterx extension
	LineCapCubic
	// rasterx extension
	LineCapQuadratic
	LineCapsN
)

//go:generate stringer -type=LineCaps

var KiT_LineCaps = kit.Enums.AddEnumAltLower(LineCapsN, kit.NotBitFlag, StylePropProps, "LineCaps")

func (ev LineCaps) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *LineCaps) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// the way in which lines are joined together: stroke-linejoin property in SVG
type LineJoins int

const (
	LineJoinMiter LineJoins = iota
	LineJoinMiterClip
	LineJoinRound
	LineJoinBevel
	LineJoinArcs
	// rasterx extension
	LineJoinArcsClip
	LineJoinsN
)

//go:generate stringer -type=LineJoins

var KiT_LineJoins = kit.Enums.AddEnumAltLower(LineJoinsN, kit.NotBitFlag, StylePropProps, "LineJoins")

func (ev LineJoins) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *LineJoins) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// IMPORTANT: any changes here must be updated below in StyleStrokeFuncs

// StrokeStyle contains all the properties for painting a line
type StrokeStyle struct {
	On         bool        `desc:"is stroke active -- if property is none then false"`
	Color      ColorSpec   `xml:"stroke" desc:"prop: stroke = stroke color specification"`
	Opacity    float32     `xml:"stroke-opacity" desc:"prop: stroke-opacity = global alpha opacity / transparency factor"`
	Width      units.Value `xml:"stroke-width" desc:"prop: stroke-width = line width"`
	MinWidth   units.Value `xml:"stroke-min-width" desc:"prop: stroke-min-width = minimum line width used for rendering -- if width is > 0, then this is the smallest line width -- this value is NOT subject to transforms so is in absolute dot values, and is ignored if vector-effects non-scaling-stroke is used -- this is an extension of the SVG / CSS standard"`
	Dashes     []float64   `xml:"stroke-dasharray" desc:"prop: stroke-dasharray = dash pattern, in terms of alternating on and off distances -- e.g., [4 4] = 4 pixels on, 4 pixels off.  Currently only supporting raw pixel numbers, but in principle should support units."`
	Cap        LineCaps    `xml:"stroke-linecap" desc:"prop: stroke-linecap = how to draw the end cap of lines"`
	Join       LineJoins   `xml:"stroke-linejoin" desc:"prop: stroke-linejoin = how to join line segments"`
	MiterLimit float32     `xml:"stroke-miterlimit" min:"1" desc:"prop: stroke-miterlimit = limit of how far to miter -- must be 1 or larger"`
}

// Defaults initializes default values for paint stroke
func (ps *StrokeStyle) Defaults() {
	ps.On = false // svg says default is off
	ps.SetColor(color.Black)
	ps.Width.Set(1.0, units.Px)
	ps.MinWidth.Set(.5, units.Dot)
	ps.Cap = LineCapButt
	ps.Join = LineJoinMiter // Miter not yet supported, but that is the default -- falls back on bevel
	ps.MiterLimit = 10.0
	ps.Opacity = 1.0
}

// SetStylePost does some updating after setting the style from user properties
func (ps *StrokeStyle) SetStylePost(props ki.Props) {
	if ps.Color.IsNil() {
		ps.On = false
	} else {
		ps.On = true
	}
}

// SetColor sets a solid stroke color -- nil turns off stroking
func (ps *StrokeStyle) SetColor(cl color.Color) {
	if cl == nil {
		ps.On = false
	} else {
		ps.On = true
		ps.Color.Color.SetColor(cl)
		ps.Color.Source = SolidColor
	}
}

// SetColorSpec sets full color spec from source
func (ps *StrokeStyle) SetColorSpec(cl *ColorSpec) {
	if cl == nil {
		ps.On = false
	} else {
		ps.On = true
		ps.Color.CopyFrom(cl)
	}
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

////////////////////////////////////////////////////////////////////////////
//   Styling functions
//     see stylefuncs.go for master version

// StyleFromProps sets style field values based on ki.Props properties
func (pc *Paint) StyleFromProps(par *Paint, props ki.Props, vp *Viewport2D) {
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		if sfunc, ok := StyleStrokeFuncs[key]; ok {
			if par != nil {
				sfunc(&pc.StrokeStyle, key, val, &par.StrokeStyle, vp)
			} else {
				sfunc(&pc.StrokeStyle, key, val, nil, vp)
			}
			continue
		}
		if sfunc, ok := StyleFillFuncs[key]; ok {
			if par != nil {
				sfunc(&pc.FillStyle, key, val, &par.FillStyle, vp)
			} else {
				sfunc(&pc.FillStyle, key, val, nil, vp)
			}
			continue
		}
		if sfunc, ok := StyleFontFuncs[key]; ok {
			if par != nil {
				sfunc(&pc.FontStyle, key, val, &par.FontStyle, vp)
			} else {
				sfunc(&pc.FontStyle, key, val, nil, vp)
			}
			continue
		}
		if sfunc, ok := StyleTextFuncs[key]; ok {
			if par != nil {
				sfunc(&pc.TextStyle, key, val, &par.TextStyle, vp)
			} else {
				sfunc(&pc.TextStyle, key, val, nil, vp)
			}
			continue
		}
		if sfunc, ok := StylePaintFuncs[key]; ok {
			sfunc(pc, key, val, par, vp)
			continue
		}
	}
}

/////////////////////////////////////////////////////////////////////////////////
//  ToDots

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (pc *Paint) ToDots(uc *units.Context) {
	pc.StyleToDots(uc)
	pc.StrokeStyle.ToDots(uc)
	pc.FillStyle.ToDots(uc)
	pc.FontStyle.ToDots(uc)
	pc.TextStyle.ToDots(uc)
}

/////////////////////////////////////////////////////////////////////////////////
//  StrokeStyle

// StyleStrokeFuncs are functions for styling the StrokeStyle object
var StyleStrokeFuncs = map[string]StyleFunc{
	"stroke": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*StrokeStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*StrokeStyle).Color
			} else if init {
				fs.Color.SetColor(color.Black)
			}
			return
		}
		fs.Color.SetIFace(val, vp, key)
	},
	"stroke-opacity": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*StrokeStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Opacity = par.(*StrokeStyle).Opacity
			} else if init {
				fs.Opacity = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			fs.Opacity = iv
		}
	},
	"stroke-width": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*StrokeStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Width = par.(*StrokeStyle).Width
			} else if init {
				fs.Width.Set(1, units.Px)
			}
			return
		}
		fs.Width.SetIFace(val, key)
	},
	"stroke-min-width": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*StrokeStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.MinWidth = par.(*StrokeStyle).MinWidth
			} else if init {
				fs.MinWidth.Set(1, units.Px)
			}
			return
		}
		fs.MinWidth.SetIFace(val, key)
	},
	"stroke-dashes": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*StrokeStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Dashes = par.(*StrokeStyle).Dashes
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
	"stroke-linecap": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*StrokeStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Cap = par.(*StrokeStyle).Cap
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
	"stroke-linejoin": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*StrokeStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Join = par.(*StrokeStyle).Join
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
	"stroke-miterlimit": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*StrokeStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.MiterLimit = par.(*StrokeStyle).MiterLimit
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

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (ss *StrokeStyle) ToDots(uc *units.Context) {
	ss.Width.ToDots(uc)
	ss.MinWidth.ToDots(uc)
}

/////////////////////////////////////////////////////////////////////////////////
//  FillStyle

// StyleFillFuncs are functions for styling the FillStyle object
var StyleFillFuncs = map[string]StyleFunc{
	"fill": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FillStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*FillStyle).Color
			} else if init {
				fs.Color.SetColor(color.Black)
			}
			return
		}
		fs.Color.SetIFace(val, vp, key)
	},
	"fill-opacity": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FillStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Opacity = par.(*FillStyle).Opacity
			} else if init {
				fs.Opacity = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			fs.Opacity = iv
		}
	},
	"fill-rule": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
		fs := obj.(*FillStyle)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Rule = par.(*FillStyle).Rule
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

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (fs *FillStyle) ToDots(uc *units.Context) {
}

/////////////////////////////////////////////////////////////////////////////////
//  PaintStyle

// StylePaintFuncs are functions for styling the StrokeStyle object
var StylePaintFuncs = map[string]StyleFunc{
	"vector-effect": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
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
	"transform": func(obj interface{}, key string, val interface{}, par interface{}, vp *Viewport2D) {
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

// StyleToDots runs ToDots on unit values, to compile down to raw pixels
func (pc *Paint) StyleToDots(uc *units.Context) {
}
