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
	"image/color"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
	"golang.org/x/net/html/charset"
)

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

// SetNormXForm sets a scaling transform to make the entire viewbox to fit the viewport
func (svg *SVG) SetNormXForm() {
	pc := &svg.Pnt
	pc.XForm = Identity2D()
	if svg.ViewBox.Size != Vec2DZero {
		// todo: deal with all the other options!
		vpsX := float32(svg.Geom.Size.X) / svg.ViewBox.Size.X
		vpsY := float32(svg.Geom.Size.Y) / svg.ViewBox.Size.Y
		svg.Pnt.XForm = svg.Pnt.XForm.Scale(vpsX, vpsY)
	}
}

// SetDPIXForm sets a scaling transform to compensate for the dpi -- svg
// rendering is done within a 96 DPI context
func (svg *SVG) SetDPIXForm() {
	pc := &svg.Pnt
	dpisc := svg.Viewport.Win.LogicalDPI() / 96.0
	pc.XForm = Scale2D(dpisc, dpisc)
}

func (svg *SVG) Init2D() {
	svg.Viewport2D.Init2D()
	bitflag.Set(&svg.Flag, int(VpFlagSVG)) // we are an svg type
	svg.Pnt.Defaults()
	svg.Pnt.FontStyle.BgColor.SetColor(color.White)
}

func (svg *SVG) Size2D() {
	svg.InitLayout2D()
	if svg.ViewBox.Size != Vec2DZero {
		svg.LayData.AllocSize = svg.ViewBox.Size
	}
	svg.Size2DAddSpace()
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
	fi, err := os.Stat(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	if fi.IsDir() {
		err := fmt.Errorf("svg.LoadXML: file is a directory: %v\n", filename)
		log.Println(err)
		return err
	}
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
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
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
			nm := se.Name.Local
			switch {
			case nm == "svg":
				if curPar != svg.This {
					curPar = curPar.AddNewChild(KiT_SVG, "svg").(Node2D)
				}
				csvg := curPar.EmbeddedStruct(KiT_SVG).(*SVG)
				curSvg = csvg
				for _, attr := range se.Attr {
					if csvg.SetStdXMLAttr(attr.Name.Local, attr.Value) {
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
			case nm == "desc":
				inDesc = true
			case nm == "title":
				inTitle = true
			case nm == "defs":
				inDef = true
				defPrevPar = curPar
				curPar = &curSvg.Defs
			case nm == "g":
				curPar = curPar.AddNewChild(KiT_SVGGroup, "g").(Node2D)
				for _, attr := range se.Attr {
					if curPar.AsNode2D().SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						curPar.SetProp(attr.Name.Local, attr.Value)
					}
				}
			case nm == "rect":
				rect := curPar.AddNewChild(KiT_Rect, "rect").(*Rect)
				var x, y, w, h, rx, ry float32
				for _, attr := range se.Attr {
					if rect.SetStdXMLAttr(attr.Name.Local, attr.Value) {
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
			case nm == "circle":
				circle := curPar.AddNewChild(KiT_Circle, "circle").(*Circle)
				var cx, cy, r float32
				for _, attr := range se.Attr {
					if circle.SetStdXMLAttr(attr.Name.Local, attr.Value) {
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
			case nm == "ellipse":
				ellipse := curPar.AddNewChild(KiT_Ellipse, "ellipse").(*Ellipse)
				var cx, cy, rx, ry float32
				for _, attr := range se.Attr {
					if ellipse.SetStdXMLAttr(attr.Name.Local, attr.Value) {
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
			case nm == "line":
				line := curPar.AddNewChild(KiT_Line, "line").(*Line)
				var x1, x2, y1, y2 float32
				for _, attr := range se.Attr {
					if line.SetStdXMLAttr(attr.Name.Local, attr.Value) {
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
			case nm == "polygon":
				polygon := curPar.AddNewChild(KiT_Polygon, "polygon").(*Polygon)
				for _, attr := range se.Attr {
					if polygon.SetStdXMLAttr(attr.Name.Local, attr.Value) {
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
			case nm == "polyline":
				polyline := curPar.AddNewChild(KiT_Polyline, "polyline").(*Polyline)
				for _, attr := range se.Attr {
					if polyline.SetStdXMLAttr(attr.Name.Local, attr.Value) {
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
			case nm == "path":
				path := curPar.AddNewChild(KiT_Path, "path").(*Path)
				for _, attr := range se.Attr {
					if path.SetStdXMLAttr(attr.Name.Local, attr.Value) {
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
			case nm == "tspan":
				fallthrough
			case nm == "text":
				var txt *SVGText
				if se.Name.Local == "text" {
					txt = curPar.AddNewChild(KiT_SVGText, "txt").(*SVGText)
					inTxt = true
					curTxt = txt
				} else {
					if inTxt && curTxt != nil {
						txt = curTxt.AddNewChild(KiT_SVGText, "tspan").(*SVGText)
						txt.Pos = curTxt.Pos
					} else {
						txt = curPar.AddNewChild(KiT_SVGText, "tspan").(*SVGText)
					}
					inTspn = true
					curTspn = txt
				}
				for _, attr := range se.Attr {
					if txt.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x":
						pts := SVGReadPoints(attr.Value)
						if len(pts) > 1 {
							txt.CharPosX = pts
						} else if len(pts) == 1 {
							txt.Pos.X = pts[0]
						}
					case "y":
						pts := SVGReadPoints(attr.Value)
						if len(pts) > 1 {
							txt.CharPosY = pts
						} else if len(pts) == 1 {
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
			case nm == "linearGradient":
				grad := curPar.AddNewChild(KiT_Gradient, "lin-grad").(*Gradient)
				for _, attr := range se.Attr {
					if grad.SetStdXMLAttr(attr.Name.Local, attr.Value) {
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
								grad.Grad.CopyFrom(&hrg.Grad)
								// fmt.Printf("successful href: %v\n", nm)
							}
						}
					}
				}
				err = grad.Grad.UnmarshalXML(decoder, se)
				if err != nil {
					return err
				}
			case nm == "radialGradient":
				grad := curPar.AddNewChild(KiT_Gradient, "rad-grad").(*Gradient)
				for _, attr := range se.Attr {
					if grad.SetStdXMLAttr(attr.Name.Local, attr.Value) {
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
								grad.Grad.CopyFrom(&hrg.Grad)
								// fmt.Printf("successful href: %v\n", nm)
							}
						}
					}
				}
				err = grad.Grad.UnmarshalXML(decoder, se)
				if err != nil {
					return err
				}
			case nm == "style":
				sty := curPar.AddNewChild(KiT_StyleSheet, "style").(*StyleSheet)
				for _, attr := range se.Attr {
					if sty.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
				}
				inCSS = true
				curCSS = sty
				// style code shows up in CharData below
			case nm == "clipPath":
				curPar = curPar.AddNewChild(KiT_ClipPath, "clip-path").(Node2D)
				cp := curPar.(*ClipPath)
				for _, attr := range se.Attr {
					if cp.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						cp.SetProp(attr.Name.Local, attr.Value)
					}
				}
			case nm == "marker":
				curPar = curPar.AddNewChild(KiT_Marker, "marker").(Node2D)
				cp := curPar.(*Marker)
				for _, attr := range se.Attr {
					if cp.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						cp.SetProp(attr.Name.Local, attr.Value)
					}
				}
			case nm == "use":
				link := XMLAttr("href", se.Attr)
				itm := curPar.FindNamedElement(link)
				if itm != nil {
					cln := itm.Clone().(Node2D)
					if cln != nil {
						curPar.AddChild(cln)
						for _, attr := range se.Attr {
							if cln.AsNode2D().SetStdXMLAttr(attr.Name.Local, attr.Value) {
								continue
							}
							switch attr.Name.Local {
							default:
								cln.SetProp(attr.Name.Local, attr.Value)
							}
						}
					}
				}
			case nm == "Work":
				fallthrough
			case nm == "RDF":
				fallthrough
			case nm == "format":
				fallthrough
			case nm == "type":
				fallthrough
			case nm == "namedview":
				fallthrough
			case nm == "perspective":
				fallthrough
			case nm == "grid":
				fallthrough
			case nm == "guide":
				fallthrough
			case nm == "metadata":
				curPar = curPar.AddNewChild(KiT_MetaData2D, nm).(Node2D)
				md := curPar.(*MetaData2D)
				md.Class = nm
				for _, attr := range se.Attr {
					if md.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						curPar.SetProp(attr.Name.Local, attr.Value)
					}
				}
			case strings.HasPrefix(nm, "flow"):
				curPar = curPar.AddNewChild(KiT_SVGFlow, nm).(Node2D)
				md := curPar.(*SVGFlow)
				md.Class = nm
				md.FlowType = nm
				for _, attr := range se.Attr {
					if md.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						curPar.SetProp(attr.Name.Local, attr.Value)
					}
				}
			case strings.HasPrefix(nm, "fe"):
			case strings.HasPrefix(nm, "filter"):
				curPar = curPar.AddNewChild(KiT_SVGFilter, nm).(Node2D)
				md := curPar.(*SVGFilter)
				md.Class = nm
				md.FilterType = nm
				for _, attr := range se.Attr {
					if md.SetStdXMLAttr(attr.Name.Local, attr.Value) {
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
			// (ok, md := curPar.(*MetaData2D); ok)
			trspc := strings.TrimSpace(string(se))
			switch {
			// case :
			// 	md.MetaData = string(se)
			case inTitle:
				curSvg.Title += trspc
			case inDesc:
				curSvg.Desc += trspc
			case inTspn && curTspn != nil:
				curTspn.Text = trspc
			case inTxt && curTxt != nil:
				curTxt.Text = trspc
			case inCSS && curCSS != nil:
				curCSS.ParseString(trspc)
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

/////////////////////////////////////////////////////////////
// SVGEdit

// SVGEdit supports editing of SVG elements
type SVGEdit struct {
	SVG
	Trans Vec2D   `desc:"view translation offset (from dragging)"`
	Scale float32 `desc:"view scaling (from zooming)"`
}

var KiT_SVGEdit = kit.Types.AddType(&SVGEdit{}, nil)

// SVGEditEvents handles svg editing events
func (svg *SVGEdit) SVGEditEvents() {
	svg.ConnectEventType(oswin.MouseDragEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		ssvg := recv.EmbeddedStruct(KiT_SVGEdit).(*SVGEdit)
		if ssvg.IsDragging() {
			del := me.Where.Sub(me.From)
			ssvg.Trans.X += float32(del.X)
			ssvg.Trans.Y += float32(del.Y)
			ssvg.SetTransform()
			ssvg.SetFullReRender()
			ssvg.UpdateSig()
		}
	})
	svg.ConnectEventType(oswin.MouseScrollEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		ssvg := recv.EmbeddedStruct(KiT_SVGEdit).(*SVGEdit)
		ssvg.InitScale()
		ssvg.Scale += float32(me.NonZeroDelta(false)) / 20
		if ssvg.Scale <= 0 {
			ssvg.Scale = 0.01
		}
		fmt.Printf("zoom: %v\n", ssvg.Scale)
		ssvg.SetTransform()
		ssvg.SetFullReRender()
		ssvg.UpdateSig()
	})
	svg.ConnectEventType(oswin.MouseEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		me.SetProcessed()
		ssvg := recv.EmbeddedStruct(KiT_SVGEdit).(*SVGEdit)
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if me.Action == mouse.Press {
			if obj != nil {
				StructViewDialog(ssvg.Viewport, obj, nil, "SVG Element View", "", nil, nil)
			}
		}
	})
	svg.ConnectEventType(oswin.MouseHoverEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.HoverEvent)
		me.SetProcessed()
		ssvg := recv.EmbeddedStruct(KiT_SVGEdit).(*SVGEdit)
		obj := ssvg.FirstContainingPoint(me.Where, true)
		if obj != nil {
			pos := me.Where
			PopupTooltip(obj.Name(), pos.X, pos.Y, svg.Viewport, obj.Name())
		}
	})
}

// InitScale ensures that Scale is initialized and non-zero
func (svg *SVGEdit) InitScale() {
	if svg.Scale == 0 {
		if svg.Viewport != nil {
			svg.Scale = svg.Viewport.Win.LogicalDPI() / 96.0
		} else {
			svg.Scale = 1
		}
	}
}

// SetTransform sets the transform based on Trans and Scale values
func (svg *SVGEdit) SetTransform() {
	svg.InitScale()
	svg.SetProp("transform", fmt.Sprintf("translate(%v,%v) scale(%v,%v)", svg.Trans.X, svg.Trans.Y, svg.Scale, svg.Scale))
}

func (svg *SVGEdit) Render2D() {
	if svg.PushBounds() {
		svg.SVGEditEvents()
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
