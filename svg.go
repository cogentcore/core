// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// svg parsing is adapted from github.com/srwiley/oksvg:
//
// Copyright 2017 The oksvg Authors. All rights reserved.
//
// created: 2/12/2017 by S.R.Wiley

package gi

import (
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
	"golang.org/x/net/html/charset"
)

////////////////////////////////////////////////////////////////////////////////////////
// SVGNodeBase

// SVGNodeBase is an element within the SVG sub-scenegraph -- does not use
// layout logic -- just renders into parent SVG viewport
type SVGNodeBase struct {
	Node2DBase
	Pnt Paint `json:"-" xml:"-" desc:"full paint information for this node"`
}

var KiT_SVGNodeBase = kit.Types.AddType(&SVGNodeBase{}, SVGNodeBaseProps)

var SVGNodeBaseProps = ki.Props{
	"base-type": true, // excludes type from user selections
}

func (g *SVGNodeBase) AsSVGNode() *SVGNodeBase {
	return g
}

// Paint satisfies the painter interface
func (g *SVGNodeBase) Paint() *Paint {
	return &g.Pnt
}

// Init2DBase handles basic node initialization -- Init2D can then do special things
func (g *SVGNodeBase) Init2DBase() {
	g.Viewport = g.ParentViewport()
	g.Pnt.Defaults()
	g.ConnectToViewport()
}

func (g *SVGNodeBase) Init2D() {
	g.Init2DBase()
}

// Style2DSVG styles the Paint values directly from node properties -- for
// SVG-style nodes -- no relevant default styling here -- parents can just set
// props directly as needed
func Style2DSVG(gii Node2D) {
	g := gii.AsNode2D()
	if g.Viewport == nil { // robust
		gii.Init2D()
	}

	pntr, ok := gii.(Painter)
	if !ok {
		return
	}
	pc := pntr.Paint()

	SetCurStyleNode2D(gii)
	defer SetCurStyleNode2D(nil)

	pc.StyleSet = false // this is always first call, restart
	var pagg *ki.Props
	pgi, pg := KiToNode2D(gii.Parent())
	if pgi != nil {
		pagg = &pg.CSSAgg
		if pp, ok := pgi.(Painter); ok {
			pc.CopyFrom(pp.Paint())
			pc.SetStyleProps(pp.Paint(), gii.Properties())
		} else {
			pc.SetStyleProps(nil, gii.Properties())
		}
	} else {
		pc.SetStyleProps(nil, gii.Properties())
	}
	// pc.SetUnitContext(g.Viewport, Vec2DZero)
	pc.ToDots() // we always inherit parent's unit context -- SVG sets it once-and-for-all
	if pagg != nil {
		AggCSS(&g.CSSAgg, *pagg)
	}
	AggCSS(&g.CSSAgg, g.CSS)
	StyleCSSSVG(gii, g.CSSAgg)
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	} else {
		pc.Off = false
	}
}

// ApplyCSSSVG applies css styles to given node, using key to select sub-props
// from overall properties list
func ApplyCSSSVG(node Node2D, key string, css ki.Props) bool {
	pntr, ok := node.(Painter)
	if !ok {
		return false
	}
	pp, got := css[key]
	if !got {
		return false
	}
	pmap, ok := pp.(ki.Props) // must be a props map
	if !ok {
		return false
	}

	pc := pntr.Paint()

	if pgi, _ := KiToNode2D(node.Parent()); pgi != nil {
		if pp, ok := pgi.(Painter); ok {
			pc.SetStyleProps(pp.Paint(), pmap)
		} else {
			pc.SetStyleProps(nil, pmap)
		}
	} else {
		pc.SetStyleProps(nil, pmap)
	}
	return true
}

// StyleCSSSVG applies css style properties to given SVG node, parsing
// out type, .class, and #name selectors
func StyleCSSSVG(node Node2D, css ki.Props) {
	tyn := strings.ToLower(node.Type().Name()) // type is most general, first
	ApplyCSSSVG(node, tyn, css)
	cln := "." + strings.ToLower(node.AsNode2D().Class) // then class
	ApplyCSSSVG(node, cln, css)
	idnm := "#" + strings.ToLower(node.Name()) // then name
	ApplyCSSSVG(node, idnm, css)
}

func (g *SVGNodeBase) Style2D() {
	Style2DSVG(g.This.(Node2D))
}

// ParentSVG returns the parent SVG viewport
func (g *SVGNodeBase) ParentSVG() *SVG {
	pvp := g.ParentViewport()
	for pvp != nil {
		if pvp.IsSVG() {
			return pvp.This.EmbeddedStruct(KiT_SVG).(*SVG)
		}
		pvp = pvp.ParentViewport()
	}
	return nil
}

func (g *SVGNodeBase) Size2D() {
}

func (g *SVGNodeBase) Layout2D(parBBox image.Rectangle) {
}

func (g *SVGNodeBase) BBox2D() image.Rectangle {
	// nodes should compute this in viewport rendering coords (not "user coords")
	// transform has already been applied
	return g.BBox
}

func (g *SVGNodeBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
}

func (g *SVGNodeBase) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

// ComputeBBoxSVG is called by default in render to compute bounding boxes for
// gui interaction -- can only be done in rendering because that is when all
// the proper xforms are all in place -- VpBBox is intersected with parent SVG
func (g *SVGNodeBase) ComputeBBoxSVG() {
	g.BBox = g.This.(Node2D).BBox2D()
	g.ObjBBox = g.BBox // no diff
	g.VpBBox = g.Viewport.VpBBox.Intersect(g.ObjBBox)
	g.SetWinBBox()
}

func (g *SVGNodeBase) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	g.ComputeBBoxSVG()
	// render goes here
	g.Render2DChildren()
	rs.PopXForm()
}

func (g *SVGNodeBase) ReRender2D() (node Node2D, layout bool) {
	svg := g.ParentSVG()
	if svg != nil {
		node = svg
	} else {
		node = g.This.(Node2D) // no other option..
	}
	layout = false
	return
}

func (g *SVGNodeBase) Move2D(delta image.Point, parBBox image.Rectangle) {
}

////////////////////////////////////////////////////////////////////////////////////////
// SVG -- the viewport

// SVG is a viewport for containing SVG drawing objects, correponding to the
// svg tag in html -- it provides its own bitmap for drawing into
type SVG struct {
	Viewport2D
	ViewBox ViewBox  `desc:"viewbox defines the coordinate system for the drawing"`
	Pnt     Paint    `json:"-" xml:"-" desc:"paint styles -- inherited by nodes"`
	Defs    SVGGroup `desc:"all defs defined elements go here (gradients, symbols, etc)"`
	Title   string   `xml:"title" desc:"the title of the svg"`
	Desc    string   `xml:"desc" desc:"the description of the svg"`
}

var KiT_SVG = kit.Types.AddType(&SVG{}, nil)

// Paint satisfies the painter interface
func (g *SVG) Paint() *Paint {
	return &g.Pnt
}

// DeleteAll deletes any existing elements in this svg
func (svg *SVG) DeleteAll() {
	updt := svg.UpdateStart()
	svg.DeleteChildren(true)
	svg.ViewBox.Defaults()
	svg.Pnt.Defaults()
	svg.Defs.DeleteChildren(true)
	svg.Title = ""
	svg.Desc = ""
	svg.UpdateEnd(updt)
}

// SetNormXForm scaling transform
func (svg *SVG) SetNormXForm() {
	svg.Pnt.XForm = Identity2D()
	if svg.ViewBox.Size != Vec2DZero {
		// todo: deal with all the other options!
		vps := NewVec2DFmPoint(svg.Geom.Size).Div(svg.ViewBox.Size)
		svg.Pnt.XForm = svg.Pnt.XForm.Scale(vps.X, vps.Y)
	}
}

func (svg *SVG) Init2D() {
	svg.Viewport2D.Init2D()
	bitflag.Set(&svg.Flag, int(VpFlagSVG)) // we are an svg type
	svg.Pnt.Defaults()
}

func (svg *SVG) Style2D() {
	svg.Style2DWidget()
	svg.Pnt.Defaults()
	Style2DSVG(svg.This.(Node2D))
	svg.Pnt.SetUnitContext(svg.AsViewport2D(), svg.ViewBox.Size) // context is viewbox
}

func (svg *SVG) Layout2D(parBBox image.Rectangle) {
	svg.Layout2DBase(parBBox, true)
	// do not call layout on children -- they don't do it
	// this is too late to affect anything
	// svg.Pnt.SetUnitContext(svg.AsViewport2D(), svg.ViewBox.Size)
}

func (svg *SVG) Render2D() {
	if svg.PushBounds() {
		rs := &svg.Render
		if svg.Fill {
			svg.FillViewport()
		}
		rs.PushXForm(svg.Pnt.XForm)
		svg.Render2DChildren() // we must do children first, then us!
		svg.PopBounds()
		rs.PopXForm()
		svg.RenderViewport2D() // update our parent image
	}
}

func (svg *SVG) FindNamedElement(name string) Node2D {
	name = strings.TrimPrefix(name, "#")
	if name == "" {
		log.Printf("gi.SVG FindNamedElement: name is empty\n")
		return nil
	}
	if svg.Nm == name {
		return svg.This.(Node2D)
	}

	def := svg.Defs.ChildByName(name, 0)
	if def != nil {
		return def.(Node2D)
	}

	if svg.Par == nil {
		log.Printf("gi.SVG FindNamedElement: could not find name: %v\n", name)
		return nil
	}
	pgi, _ := KiToNode2D(svg.Par)
	if pgi != nil {
		return pgi.FindNamedElement(name)
	}
	log.Printf("gi.SVG FindNamedElement: could not find name: %v\n", name)
	return nil
}

/////////////////////////////////////////////////////////////////////////////////
//   SVG IO

// see https://github.com/goki/ki/wiki/Naming for IO naming conventions
// using standard XML marshal / unmarshal

var (
	paramMismatchError  = errors.New("gi.SVG Parse: Param mismatch")
	commandUnknownError = errors.New("gi.SVG Parse: Unknown command")
	zeroLengthIdError   = errors.New("gi.SVG Parse: zero length id")
	missingIdError      = errors.New("gi.SVG Parse: cannot find id")
)

// SVGParseFloat32 logs any strconv.ParseFloat errors
func SVGParseFloat32(pstr string) (float32, error) {
	r, err := strconv.ParseFloat(pstr, 32)
	if err != nil {
		log.Printf("gi.SVGParseFloat32: error parsing float32 number from: %v, %v\n", pstr, err)
		return float32(0.0), err
	}
	return float32(r), nil
}

// SVGReadPoints reads a set of floating point values from a SVG format number
// string -- returns a slice or nil if there was an error
func SVGReadPoints(pstr string) []float32 {
	lastIdx := -1
	var pts []float32
	lr := ' '
	for i, r := range pstr {
		if unicode.IsNumber(r) == false && r != '.' && !(r == '-' && lr == 'e') && r != 'e' {
			if lastIdx != -1 {
				s := pstr[lastIdx:i]
				p, err := SVGParseFloat32(s)
				if err != nil {
					return nil
				}
				pts = append(pts, p)
			}
			if r == '-' {
				lastIdx = i
			} else {
				lastIdx = -1
			}
		} else if lastIdx == -1 {
			lastIdx = i
		}
		lr = r
	}
	if lastIdx != -1 && lastIdx != len(pstr) {
		s := pstr[lastIdx:len(pstr)]
		p, err := SVGParseFloat32(s)
		if err != nil {
			return nil
		}
		pts = append(pts, p)
	}
	return pts
}

// SVGPointsCheckN checks the number of points read and emits an error if not equal to n
func SVGPointsCheckN(pts []float32, n int, errmsg string) error {
	if len(pts) != n {
		return fmt.Errorf("%v incorrect number of points: %v != %v\n", errmsg, len(pts), n)
	}
	return nil
}

// XMLAttr searches for given attribute in slice of xml attributes -- returns "" if not found
func XMLAttr(name string, attrs []xml.Attr) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

// LoadXML Loads XML-formatted SVG input from given file
func (svg *SVG) LoadXML(filename string) error {
	fp, err := os.Open(filename)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	return svg.ReadXML(fp)
}

// ReadXML reads XML-formatted SVG input from io.Reader, and uses
// xml.Decoder to create the SVG scenegraph for corresponding SVG drawing.
// Removes any existing content in SVG first. To process a byte slice, pass:
// bytes.NewReader([]byte(str)) -- all errors are logged and also returned.
func (svg *SVG) ReadXML(reader io.Reader) error {
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel
	for {
		t, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("gi.SVG parsing error: %v\n", err)
			return err
		}
		switch se := t.(type) {
		case xml.StartElement:
			return svg.UnmarshalXML(decoder, se)
			// todo: ignore rest?
		}
	}
	return nil
}

// UnmarshalXML unmarshals the svg using xml.Decoder
func (svg *SVG) UnmarshalXML(decoder *xml.Decoder, se xml.StartElement) error {
	updt := svg.UpdateStart()
	defer svg.UpdateEnd(updt)

	start := &se

	svg.DeleteAll()

	curPar := svg.This.(Node2D) // current parent node into which elements are created
	curSvg := svg
	inTitle := false
	inDesc := false
	inDef := false
	inCSS := false
	var curCSS *StyleSheet
	inTxt := false
	var curTxt *SVGText
	inTspn := false
	var curTspn *SVGText
	var defPrevPar Node2D // previous parent before a def encountered

	for {
		var t xml.Token
		var err error
		if start != nil {
			t = *start
			start = nil
		} else {
			t, err = decoder.Token()
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("gi.SVG parsing error: %v\n", err)
			return err
		}
		switch se := t.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "svg":
				if curPar != svg.This {
					curPar = curPar.AddNewChild(KiT_SVG, "svg").(Node2D)
				}
				csvg := curPar.EmbeddedStruct(KiT_SVG).(*SVG)
				curSvg = csvg
				for _, attr := range se.Attr {
					if csvg.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "viewBox":
						pts := SVGReadPoints(attr.Value)
						if len(pts) != 4 {
							return paramMismatchError
						}
						csvg.ViewBox.Min.X = pts[0]
						csvg.ViewBox.Min.Y = pts[1]
						csvg.ViewBox.Size.X = pts[2]
						csvg.ViewBox.Size.Y = pts[3]
						// csvg.SetProp("width", units.NewValue(float32(csvg.ViewBox.Size.X), units.Dot))
						// csvg.SetProp("height", units.NewValue(float32(csvg.ViewBox.Size.Y), units.Dot))
					case "width":
						wd := units.Value{}
						wd.SetString(attr.Value)
						wd.ToDots(&csvg.Pnt.UnContext)
						csvg.ViewBox.Size.X = wd.Dots
					case "height":
						ht := units.Value{}
						ht.SetString(attr.Value)
						ht.ToDots(&csvg.Pnt.UnContext)
						csvg.ViewBox.Size.Y = ht.Dots
					default:
						curPar.SetProp(attr.Name.Local, attr.Value)
					}
				}
			case "desc":
				inDesc = true
			case "title":
				inTitle = true
			case "defs":
				inDef = true
				defPrevPar = curPar
				curPar = &curSvg.Defs
			case "g":
				curPar = curPar.AddNewChild(KiT_SVGGroup, "g").(Node2D)
				for _, attr := range se.Attr {
					if curPar.AsNode2D().SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						curPar.SetProp(attr.Name.Local, attr.Value)
					}
				}
			case "rect":
				rect := curPar.AddNewChild(KiT_Rect, "rect").(*Rect)
				var x, y, w, h, rx, ry float32
				for _, attr := range se.Attr {
					if rect.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x":
						x, err = SVGParseFloat32(attr.Value)
					case "y":
						y, err = SVGParseFloat32(attr.Value)
					case "width":
						w, err = SVGParseFloat32(attr.Value)
					case "height":
						h, err = SVGParseFloat32(attr.Value)
					case "rx":
						rx, err = SVGParseFloat32(attr.Value)
					case "ry":
						ry, err = SVGParseFloat32(attr.Value)
					default:
						rect.SetProp(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
				rect.Pos.Set(x, y)
				rect.Size.Set(w, h)
				rect.Radius.Set(rx, ry)
			case "circle":
				circle := curPar.AddNewChild(KiT_Circle, "circle").(*Circle)
				var cx, cy, r float32
				for _, attr := range se.Attr {
					if circle.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "cx":
						cx, err = SVGParseFloat32(attr.Value)
					case "cy":
						cy, err = SVGParseFloat32(attr.Value)
					case "r":
						r, err = SVGParseFloat32(attr.Value)
					default:
						circle.SetProp(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
				circle.Pos.Set(cx, cy)
				circle.Radius = r
			case "ellipse":
				ellipse := curPar.AddNewChild(KiT_Ellipse, "ellipse").(*Ellipse)
				var cx, cy, rx, ry float32
				for _, attr := range se.Attr {
					if ellipse.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "cx":
						cx, err = SVGParseFloat32(attr.Value)
					case "cy":
						cy, err = SVGParseFloat32(attr.Value)
					case "rx":
						rx, err = SVGParseFloat32(attr.Value)
					case "ry":
						ry, err = SVGParseFloat32(attr.Value)
					default:
						ellipse.SetProp(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
				ellipse.Pos.Set(cx, cy)
				ellipse.Radii.Set(rx, ry)
			case "line":
				line := curPar.AddNewChild(KiT_Line, "line").(*Line)
				var x1, x2, y1, y2 float32
				for _, attr := range se.Attr {
					if line.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x1":
						x1, err = SVGParseFloat32(attr.Value)
					case "y1":
						y1, err = SVGParseFloat32(attr.Value)
					case "x2":
						x2, err = SVGParseFloat32(attr.Value)
					case "y2":
						y2, err = SVGParseFloat32(attr.Value)
					default:
						line.SetProp(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
				line.Start.Set(x1, y1)
				line.End.Set(x2, y2)
			case "polygon":
				polygon := curPar.AddNewChild(KiT_Polygon, "polygon").(*Polygon)
				for _, attr := range se.Attr {
					if polygon.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "points":
						pts := SVGReadPoints(attr.Value)
						if pts != nil {
							sz := len(pts)
							if sz%2 != 0 {
								err = fmt.Errorf("gi.SVG polygon has an odd number of points: %v str: %v\n", sz, attr.Value)
								log.Println(err)
								return err
							}
							pvec := make([]Vec2D, sz/2)
							for ci := 0; ci < sz/2; ci++ {
								pvec[ci].Set(pts[ci*2], pts[ci*2+1])
							}
							polygon.Points = pvec
						}
					default:
						polygon.SetProp(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
			case "polyline":
				polyline := curPar.AddNewChild(KiT_Polyline, "polyline").(*Polyline)
				for _, attr := range se.Attr {
					if polyline.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "points":
						pts := SVGReadPoints(attr.Value)
						if pts != nil {
							sz := len(pts)
							if sz%2 != 0 {
								err = fmt.Errorf("gi.SVG polyline has an odd number of points: %v str: %v\n", sz, attr.Value)
								log.Println(err)
								return err
							}
							pvec := make([]Vec2D, sz/2)
							for ci := 0; ci < sz/2; ci++ {
								pvec[ci].Set(pts[ci*2], pts[ci*2+1])
							}
							polyline.Points = pvec
						}
					default:
						polyline.SetProp(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
			case "path":
				path := curPar.AddNewChild(KiT_Path, "path").(*Path)
				for _, attr := range se.Attr {
					if path.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "d":
						path.SetData(attr.Value)
					default:
						path.SetProp(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
			case "tspan":
				fallthrough
			case "text":
				var txt *SVGText
				if se.Name.Local == "text" {
					txt = curPar.AddNewChild(KiT_SVGText, "txt").(*SVGText)
					inTxt = true
					curTxt = txt
				} else {
					if inTxt && curTxt != nil {
						txt = curTxt.AddNewChild(KiT_SVGText, "tspan").(*SVGText)
					} else {
						txt = curPar.AddNewChild(KiT_SVGText, "tspan").(*SVGText)
					}
					inTspn = true
					curTspn = txt
				}
				for _, attr := range se.Attr {
					if txt.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x":
						pts := SVGReadPoints(attr.Value)
						if len(pts) > 1 {
							txt.CharPosX = pts
						}
						if len(pts) > 0 {
							txt.Pos.X = pts[0]
						}
					case "y":
						pts := SVGReadPoints(attr.Value)
						if len(pts) > 1 {
							txt.CharPosY = pts
						}
						if len(pts) > 0 {
							txt.Pos.Y = pts[0]
						}
					case "dx":
						pts := SVGReadPoints(attr.Value)
						if len(pts) > 0 {
							txt.CharPosDX = pts
						}
					case "dy":
						pts := SVGReadPoints(attr.Value)
						if len(pts) > 0 {
							txt.CharPosDY = pts
						}
					case "rotate":
						pts := SVGReadPoints(attr.Value)
						if len(pts) > 0 {
							txt.CharRots = pts
						}
					case "textLength":
						tl, err := SVGParseFloat32(attr.Value)
						if err != nil {
							txt.TextLength = tl
						}
					case "lengthAdjust":
						if attr.Value == "spacingAndGlyphs" {
							txt.AdjustGlyphs = true
						} else {
							txt.AdjustGlyphs = false
						}
					default:
						txt.SetProp(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
			case "linearGradient":
				grad := curPar.AddNewChild(KiT_Gradient, "lin-grad").(*Gradient)
				for _, attr := range se.Attr {
					if grad.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "href":
						nm := attr.Value
						if strings.HasPrefix(nm, "#") {
							nm = strings.TrimPrefix(nm, "#")
						}
						hr := curPar.ChildByName(nm, 0)
						if hr != nil {
							if hrg, ok := hr.(*Gradient); ok {
								grad.Grad = hrg.Grad
								// fmt.Printf("successful href: %v\n", nm)
							}
						}
					}
				}
				err = grad.Grad.UnmarshalXML(decoder, se)
				if err != nil {
					return err
				}
			case "radialGradient":
				grad := curPar.AddNewChild(KiT_Gradient, "rad-grad").(*Gradient)
				for _, attr := range se.Attr {
					if grad.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "href":
						nm := attr.Value
						if strings.HasPrefix(nm, "#") {
							nm = strings.TrimPrefix(nm, "#")
						}
						hr := curPar.ChildByName(nm, 0)
						if hr != nil {
							if hrg, ok := hr.(*Gradient); ok {
								grad.Grad = hrg.Grad
								// fmt.Printf("successful href: %v\n", nm)
							}
						}
					}
				}
				err = grad.Grad.UnmarshalXML(decoder, se)
				if err != nil {
					return err
				}
			case "style":
				sty := curPar.AddNewChild(KiT_StyleSheet, "style").(*StyleSheet)
				for _, attr := range se.Attr {
					if sty.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
				}
				inCSS = true
				curCSS = sty
				// style code shows up in CharData below
			case "clipPath":
				curPar = curPar.AddNewChild(KiT_ClipPath, "clip-path").(Node2D)
				cp := curPar.(*ClipPath)
				for _, attr := range se.Attr {
					if cp.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						cp.SetProp(attr.Name.Local, attr.Value)
					}
				}
			case "use":
				link := XMLAttr("href", se.Attr)
				itm := curPar.FindNamedElement(link)
				if itm != nil {
					cln := itm.Clone().(Node2D)
					if cln != nil {
						curPar.AddChild(cln)
						for _, attr := range se.Attr {
							if cln.AsNode2D().SetStdAttr(attr.Name.Local, attr.Value) {
								continue
							}
							switch attr.Name.Local {
							default:
								cln.SetProp(attr.Name.Local, attr.Value)
							}
						}
					}
				}
			case "Work":
				fallthrough
			case "RDF":
				fallthrough
			case "format":
				fallthrough
			case "type":
				fallthrough
			case "namedview":
				fallthrough
			case "perspective":
				fallthrough
			case "grid":
				fallthrough
			case "guide":
				fallthrough
			case "metadata":
				curPar = curPar.AddNewChild(KiT_MetaData2D, "metadata").(Node2D)
				md := curPar.(*MetaData2D)
				md.Class = se.Name.Local
				for _, attr := range se.Attr {
					if md.SetStdAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						curPar.SetProp(attr.Name.Local, attr.Value)
					}
				}
			default:
				errStr := "gi.SVG Cannot process svg element " + se.Name.Local
				log.Println(errStr)
				IconAutoLoad = false
			}
		case xml.EndElement:
			switch se.Name.Local {
			case "title":
				inTitle = false
			case "desc":
				inDesc = false
			case "style":
				inCSS = false
				curCSS = nil
			case "text":
				inTxt = false
				curTxt = nil
			case "tspan":
				inTspn = false
				curTspn = nil
			case "defs":
				if inDef {
					inDef = false
					curPar = defPrevPar
				}
			case "rect":
			case "circle":
			case "ellipse":
			case "line":
			case "polygon":
			case "polyline":
			case "path":
			case "use":
			case "linearGradient":
			case "radialGradient":
			default:
				if curPar == svg.This {
					break
				}
				if curPar.Parent() == nil {
					break
				}
				curPar = curPar.Parent().(Node2D)
				if curPar == svg.This {
					break
				}
				curSvg = curPar.ParentByType(KiT_SVG, true).EmbeddedStruct(KiT_SVG).(*SVG)
			}
		case xml.CharData:
			switch {
			case inTitle:
				curSvg.Title += string(se)
			case inDesc:
				curSvg.Desc += string(se)
			case inTspn && curTspn != nil:
				curTspn.Text = string(se)
			case inTxt && curTxt != nil:
				curTxt.Text = string(se)
			case inCSS && curCSS != nil:
				curCSS.ParseString(string(se))
				cp := curCSS.CSSProps()
				if cp != nil {
					if inDef && defPrevPar != nil {
						defPrevPar.AsNode2D().CSS = cp
					} else {
						curPar.AsNode2D().CSS = cp
					}
				}
			}
		}
	}
	return nil
}
