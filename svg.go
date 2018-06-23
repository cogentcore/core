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
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
	"golang.org/x/net/html/charset"
)

// SVG is a viewport for containing SVG drawing objects, correponding to the
// svg tag in html -- it provides its own bitmap for drawing into
type SVG struct {
	Viewport2D
	ViewBox ViewBox `desc:"viewbox defines the coordinate system for the drawing"`
	Defs    Group2D `desc:"all defs defined elements go here (gradients, symbols, etc)"`
	Title   string  `xml:"title" desc:"the title of the svg"`
	Desc    string  `xml:"desc" desc:"the description of the svg"`
}

var KiT_SVG = kit.Types.AddType(&SVG{}, nil)

// DeleteAll deletes any existing elements in this svg
func (svg *SVG) DeleteAll() {
	updt := svg.UpdateStart()
	svg.DeleteChildren(true)
	svg.Defs.DeleteChildren(true)
	svg.Title = ""
	svg.Desc = ""
	svg.ViewBox.Defaults()
	svg.UpdateEnd(updt)

}

// SetNormXForm scaling transform
func (svg *SVG) SetNormXForm() {
	pc := &svg.Paint
	pc.Identity()
	if svg.ViewBox.Size != Vec2DZero {
		vps := NewVec2DFmPoint(svg.Geom.Size).Div(svg.ViewBox.Size)
		pc.Scale(vps.X, vps.Y)
	}
}

func (svg *SVG) Init2D() {
	svg.Viewport2D.Init2D()
	bitflag.Set(&svg.Flag, int(VpFlagSVG)) // we are an svg type
}

func (svg *SVG) Style2D() {
	svg.Style2DWidget()
	svg.Style2DSVG() // this must come second
}

func (svg *SVG) Layout2D(parBBox image.Rectangle) {
	pc := &svg.Paint
	rs := &svg.Render
	svg.Layout2DBase(parBBox, true)
	rs.PushXForm(pc.XForm) // need xforms to get proper bboxes during layout
	svg.Layout2DChildren()
	rs.PopXForm()
}

func (svg *SVG) Render2D() {
	if svg.PushBounds() {
		pc := &svg.Paint
		rs := &svg.Render
		if svg.Fill {
			svg.FillViewport()
		}
		rs.PushXForm(pc.XForm)
		svg.Render2DChildren() // we must do children first, then us!
		svg.PopBounds()
		rs.PopXForm()
		svg.RenderViewport2D() // update our parent image
	}
}

func (svg *SVG) FindNamedElement(name string) Node2D {
	if svg.Nm == name {
		return svg.This.(Node2D)
	}

	def := svg.Defs.ChildByName(name, 0)
	if def != nil {
		return def.(Node2D)
	}

	if svg.Par == nil {
		return nil
	}
	pgi, _ := KiToNode2D(svg.Par)
	if pgi != nil {
		return pgi.FindNamedElement(name)
	}
	return nil
}

/////////////////////////////////////////////////////////////////////////////////
// Misc SVG-specific nodes

// Gradient is used for holding a specified color gradient -- name is id for
// lookup in url
type Gradient struct {
	Node2DBase
	Grad ColorSpec `desc:"the color gradient"`
}

var KiT_Gradient = kit.Types.AddType(&Gradient{}, nil)

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
						csvg.SetProp("width", units.NewValue(float32(csvg.ViewBox.Size.X), units.Dot))
						csvg.SetProp("height", units.NewValue(float32(csvg.ViewBox.Size.Y), units.Dot))
					case "width":
						csvg.SetProp("width", attr.Value)
					case "height":
						csvg.SetProp("height", attr.Value)
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
				curPar = curPar.AddNewChild(KiT_Group2D, "g").(Node2D)
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
			default:
				errStr := "gi.SVG Cannot process svg element " + se.Name.Local
				log.Println(errStr)
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
			case "defs":
				if inDef {
					inDef = false
					curPar = defPrevPar
				}
			case "g":
				fallthrough
			case "svg":
				if curPar == svg.This {
					break
				}
				curPar = curPar.Parent().(Node2D)
				if curPar == svg.This {
					break
				}
				curSvg = curPar.ParentByType(KiT_SVG, true).EmbeddedStruct(KiT_SVG).(*SVG)
			default:
			}
		case xml.CharData:
			if inTitle {
				curSvg.Title += string(se)
			}
			if inDesc {
				curSvg.Desc += string(se)
			}
			if inCSS && curCSS != nil {
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
