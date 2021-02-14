// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"image"
	"image/color"

	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Painter defines an interface for anything that has a Paint style on it
type Painter interface {
	Paint() *Paint
}

// Paint provides the styling parameters for rendering
type Paint struct {
	Off         bool          `desc:"node and everything below it are off, non-rendering"`
	StrokeStyle Stroke        `desc:"stroke (line drawing) parameters"`
	FillStyle   Fill          `desc:"fill (region filling) parameters"`
	FontStyle   Font          `desc:"font also has global opacity setting, along with generic color, background-color settings, which can be copied into stroke / fill as needed"`
	TextStyle   Text          `desc:"font also has global opacity setting, along with generic color, background-color settings, which can be copied into stroke / fill as needed"`
	VecEff      VectorEffects `xml:"vector-effect" desc:"prop: vector-effect = various rendering special effects settings"`
	XForm       mat32.Mat2    `xml:"transform" desc:"prop: transform = our additions to transform -- pushed to render state"`
	UnContext   units.Context `xml:"-" desc:"units context -- parameters necessary for anchoring relative units"`
	StyleSet    bool          `desc:"have the styles already been set?"`
	PropsNil    bool          `desc:"set to true if parent node has no props -- allows optimization of styling"`
	dotsSet     bool
	lastUnCtxt  units.Context
}

func (pc *Paint) Defaults() {
	pc.Off = false
	pc.StyleSet = false
	pc.StrokeStyle.Defaults()
	pc.FillStyle.Defaults()
	pc.FontStyle.Defaults()
	pc.TextStyle.Defaults()
	pc.XForm = mat32.Identity2D()
}

// CopyStyleFrom copies styles from another paint
func (pc *Paint) CopyStyleFrom(cp *Paint) {
	pc.Off = cp.Off
	pc.UnContext = cp.UnContext
	pc.StrokeStyle = cp.StrokeStyle
	pc.FillStyle = cp.FillStyle
	pc.FontStyle = cp.FontStyle
	pc.TextStyle = cp.TextStyle
	pc.VecEff = cp.VecEff
}

// InheritFields from parent: Manual inheriting of values is much faster than
// automatic version!
func (pc *Paint) InheritFields(par *Paint) {
	pc.FontStyle.InheritFields(&par.FontStyle)
	pc.TextStyle.InheritFields(&par.TextStyle)
}

// SetStyleProps sets paint values based on given property map (name: value
// pairs), inheriting elements as appropriate from parent, and also having a
// default style for the "initial" setting
func (pc *Paint) SetStyleProps(par *Paint, props ki.Props, ctxt Context) {
	if !pc.StyleSet && par != nil { // first time
		pc.InheritFields(par)
	}
	pc.StyleFromProps(par, props, ctxt)

	pc.StrokeStyle.SetStylePost(props)
	pc.FillStyle.SetStylePost(props)
	pc.FontStyle.SetStylePost(props)
	pc.TextStyle.SetStylePost(props)
	pc.PropsNil = (len(props) == 0)
	pc.StyleSet = true
}

// StyleToDots runs ToDots on unit values, to compile down to raw pixels
func (pc *Paint) StyleToDots(uc *units.Context) {
}

// ToDotsImpl runs ToDots on unit values, to compile down to raw pixels
func (pc *Paint) ToDotsImpl(uc *units.Context) {
	pc.StyleToDots(uc)
	pc.StrokeStyle.ToDots(uc)
	pc.FillStyle.ToDots(uc)
	pc.FontStyle.ToDots(uc)
	pc.TextStyle.ToDots(uc)
}

// SetUnitContextExt sets the unit context for external usage of paint
// outside of a Viewport, based on overall size of painting canvas.
// caches everything out in terms of raw pixel dots for rendering
// call at start of render.
func (pc *Paint) SetUnitContextExt(size image.Point) {
	pc.UnContext.Defaults()
	pc.UnContext.DPI = 96 // paint (SVG) context is always 96 = 1to1
	pc.UnContext.SetSizes(float32(size.X), float32(size.Y), float32(size.X), float32(size.Y))
	pc.FontStyle.SetUnitContext(&pc.UnContext)
	pc.ToDotsImpl(&pc.UnContext)
	pc.dotsSet = true
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (pc *Paint) ToDots() {
	if !(pc.dotsSet && pc.UnContext == pc.lastUnCtxt && pc.PropsNil) {
		pc.ToDotsImpl(&pc.UnContext)
		pc.dotsSet = true
		pc.lastUnCtxt = pc.UnContext
	}
}

//////////////////////////////////////////////////////////////////////////////////
// State query

// does the current Paint have an active stroke to render?
func (pc *Paint) HasStroke() bool {
	return pc.StrokeStyle.On
}

// does the current Paint have an active fill to render?
func (pc *Paint) HasFill() bool {
	return pc.FillStyle.On
}

// does the current Paint not have either a stroke or fill?  in which case, often we just skip it
func (pc *Paint) HasNoStrokeOrFill() bool {
	return (!pc.StrokeStyle.On && !pc.FillStyle.On)
}

/////////////////////////////////////////////////////////////////
//  enums

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

// VectorEffects contains special effects for rendering
type VectorEffects int32

const (
	VecEffNone VectorEffects = iota

	// VecEffNonScalingStroke means that the stroke width is not affected by
	// transform properties
	VecEffNonScalingStroke

	VecEffN
)

//go:generate stringer -type=VectorEffects

var KiT_VectorEffects = kit.Enums.AddEnumAltLower(VecEffN, kit.NotBitFlag, StylePropProps, "VecEff")

func (ev VectorEffects) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *VectorEffects) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// IMPORTANT: any changes here must be updated below in StyleFillFuncs

// Fill contains all the properties for filling a region
type Fill struct {
	On      bool      `desc:"is fill active -- if property is none then false"`
	Color   ColorSpec `xml:"fill" desc:"prop: fill = fill color specification"`
	Opacity float32   `xml:"fill-opacity" desc:"prop: fill-opacity = global alpha opacity / transparency factor"`
	Rule    FillRules `xml:"fill-rule" desc:"prop: fill-rule = rule for how to fill more complex shapes with crossing lines"`
}

// Defaults initializes default values for paint fill
func (pf *Fill) Defaults() {
	pf.On = true // svg says fill is ON by default
	pf.SetColor(color.Black)
	pf.Rule = FillRuleNonZero
	pf.Opacity = 1.0
}

// SetStylePost does some updating after setting the style from user properties
func (pf *Fill) SetStylePost(props ki.Props) {
	if pf.Color.IsNil() {
		pf.On = false
	} else {
		pf.On = true
	}
}

// SetColor sets a solid fill color -- nil turns off filling
func (pf *Fill) SetColor(cl color.Color) {
	if cl == nil {
		pf.On = false
	} else {
		pf.On = true
		pf.Color.Color.SetColor(cl)
		pf.Color.Source = SolidColor
	}
}

// SetColorSpec sets full color spec from source
func (pf *Fill) SetColorSpec(cl *ColorSpec) {
	if cl == nil {
		pf.On = false
	} else {
		pf.On = true
		pf.Color.CopyFrom(cl)
	}
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (fs *Fill) ToDots(uc *units.Context) {
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

// Stroke contains all the properties for painting a line
type Stroke struct {
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
func (ps *Stroke) Defaults() {
	ps.On = false // svg says default is off
	ps.SetColor(Black)
	ps.Width.Set(1.0, units.Px)
	ps.MinWidth.Set(.5, units.Dot)
	ps.Cap = LineCapButt
	ps.Join = LineJoinMiter // Miter not yet supported, but that is the default -- falls back on bevel
	ps.MiterLimit = 10.0
	ps.Opacity = 1.0
}

// SetStylePost does some updating after setting the style from user properties
func (ps *Stroke) SetStylePost(props ki.Props) {
	if ps.Color.IsNil() {
		ps.On = false
	} else {
		ps.On = true
	}
}

// SetColor sets a solid stroke color -- nil turns off stroking
func (ps *Stroke) SetColor(cl color.Color) {
	if cl == nil {
		ps.On = false
	} else {
		ps.On = true
		ps.Color.Color.SetColor(cl)
		ps.Color.Source = SolidColor
	}
}

// SetColorSpec sets full color spec from source
func (ps *Stroke) SetColorSpec(cl *ColorSpec) {
	if cl == nil {
		ps.On = false
	} else {
		ps.On = true
		ps.Color.CopyFrom(cl)
	}
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (ss *Stroke) ToDots(uc *units.Context) {
	ss.Width.ToDots(uc)
	ss.MinWidth.ToDots(uc)
}
