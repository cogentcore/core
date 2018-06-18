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
	"strconv"
	"strings"
	"unicode"

	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
	"golang.org/x/net/html/charset"
)

// SVG is a viewport for containing SVG drawing objects, correponding to the
// svg tag in html -- it provides its own bitmap for drawing into
type SVG struct {
	Viewport2D
	Defs  Group2D `desc:"all defs defined elements go here (gradients, symbols, etc)"`
	Title string  `xml:"title" desc:"the title of the svg"`
	Desc  string  `xml:"desc" desc:"the description of the svg"`
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

// SetNormXForm sets a normalized 0-1 scaling transform so svg's use 0-1
// coordinates that map to actual size of the viewport -- used e.g. for Icon
func (svg *SVG) SetNormXForm() {
	pc := &svg.Paint
	pc.Identity()
	vps := Vec2D{}
	vps.SetPoint(svg.ViewBox.Size)
	pc.Scale(vps.X, vps.Y)
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

/////////////////////////////////////////////////////////////////////////////////
//   Parsing

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
					log.Printf("gi.ReadPoints error parsing float32 number from: %v, %v\n", s, err)
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
			log.Printf("gi.ReadPoints error parsing float32 number from: %v, %v\n", s, err)
			return nil
		}
		pts = append(pts, p)
	}
	return pts
}

// SVGSetStyle sets style attributes from svg style string
func SVGSetStyle(el ki.Ki, style string) {
	st := strings.Split(style, ";")
	for _, s := range st {
		kv := strings.Split(s, ":")
		if len(kv) >= 2 {
			k := strings.TrimSpace(strings.ToLower(kv[0]))
			v := strings.TrimSpace(kv[1])
			el.SetProp(k, v)
		}
	}
}

// ParseXML parses the given XML-formatted SVG input, creating the
// corresponding SVG drawing -- removes any existing content in SVG first --
// to process a byte slice, pass: bytes.NewReader([]byte(str)) -- all errors
// are logged and also returned
func (svg *SVG) ParseXML(fin io.Reader) error {
	updt := svg.UpdateStart()
	defer svg.UpdateEnd(updt)

	svg.DeleteAll()

	curPar := svg.This // current parent node into which elements are created
	curSvg := svg
	inTitle := false
	inDesc := false
	inDef := false
	inGrad := false
	var defPrevPar ki.Ki // previous parent before a def encountered

	decoder := xml.NewDecoder(fin)
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
			switch se.Name.Local {
			case "svg":
				if curPar != svg.This {
					curPar = curPar.AddNewChild(KiT_SVG, "svg")
				}
				csvg := curPar.(*SVG)
				curSvg = csvg
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					case "viewBox":
						pts := SVGReadPoints(attr.Value)
						if len(pts) != 4 {
							return paramMismatchError
						}
						csvg.ViewBox.Min.X = int(pts[0])
						csvg.ViewBox.Min.Y = int(pts[1])
						csvg.ViewBox.Size.X = int(pts[2])
						csvg.ViewBox.Size.Y = int(pts[3])
						csvg.SetProp("width", units.NewValue(float32(csvg.ViewBox.Size.X), units.Dot))
						csvg.SetProp("height", units.NewValue(float32(csvg.ViewBox.Size.Y), units.Dot))
					case "width":
						csvg.SetProp("width", attr.Value)
					case "height":
						csvg.SetProp("height", attr.Value)
					case "id":
						curPar.SetName(attr.Value)
					case "style":
						SVGSetStyle(curPar, attr.Value)
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
				curPar = &svg.Defs
			case "g":
				curPar := curPar.AddNewChild(KiT_Group2D, "g")
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					case "id":
						curPar.SetName(attr.Value)
					case "style":
						SVGSetStyle(curPar, attr.Value)
					default:
						curPar.SetProp(attr.Name.Local, attr.Value)
					}
				}
			case "rect":
				rect := curPar.AddNewChild(KiT_Rect, "rect").(*Rect)
				var x, y, w, h, rx, ry float32
				for _, attr := range se.Attr {
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
					case "id":
						rect.SetName(attr.Value)
					case "style":
						SVGSetStyle(rect.This, attr.Value)
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
					switch attr.Name.Local {
					case "cx":
						cx, err = SVGParseFloat32(attr.Value)
					case "cy":
						cy, err = SVGParseFloat32(attr.Value)
					case "r":
						r, err = SVGParseFloat32(attr.Value)
					case "id":
						circle.SetName(attr.Value)
					case "style":
						SVGSetStyle(circle.This, attr.Value)
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
					switch attr.Name.Local {
					case "cx":
						cx, err = SVGParseFloat32(attr.Value)
					case "cy":
						cy, err = SVGParseFloat32(attr.Value)
					case "rx":
						rx, err = SVGParseFloat32(attr.Value)
					case "ry":
						ry, err = SVGParseFloat32(attr.Value)
					case "id":
						ellipse.SetName(attr.Value)
					case "style":
						SVGSetStyle(ellipse.This, attr.Value)
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
					switch attr.Name.Local {
					case "x1":
						x1, err = SVGParseFloat32(attr.Value)
					case "y1":
						y1, err = SVGParseFloat32(attr.Value)
					case "x2":
						x2, err = SVGParseFloat32(attr.Value)
					case "y2":
						y2, err = SVGParseFloat32(attr.Value)
					case "id":
						line.SetName(attr.Value)
					case "style":
						SVGSetStyle(line.This, attr.Value)
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
					case "id":
						polygon.SetName(attr.Value)
					case "style":
						SVGSetStyle(polygon.This, attr.Value)
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
					case "id":
						polyline.SetName(attr.Value)
					case "style":
						SVGSetStyle(polyline.This, attr.Value)
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
					switch attr.Name.Local {
					case "d":
						path.SetData(attr.Value)
					case "id":
						path.SetName(attr.Value)
					case "style":
						SVGSetStyle(path.This, attr.Value)
					default:
						path.SetProp(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
			// case "linearGradient":
			// 	cs.Gradient = &rasterx.Gradient{Points: [5]float32{0, 0, 0, 1, 0},
			// 		IsRadial: false, Matrix: rasterx.Identity}
			// 	for _, attr := range se.Attr {
			// 		switch attr.Name.Local {
			// 		case "id":
			// 			// id := attr.Value
			// 			// if len(id) >= 0 {
			// 			// 	svg.Ids[id] = cursor.grad
			// 			// } else {
			// 			// 	return icon, zeroLengthIdError
			// 			// }
			// 		case "x1":
			// 			cs.Gradient.Points[0], err = readFraction(attr.Value)
			// 		case "y1":
			// 			cs.Gradient.Points[1], err = readFraction(attr.Value)
			// 		case "x2":
			// 			cs.Gradient.Points[2], err = readFraction(attr.Value)
			// 		case "y2":
			// 			cs.Gradient.Points[3], err = readFraction(attr.Value)
			// 		default:
			// 			err = cs.ReadGradAttr(attr)
			// 		}
			// 		if err != nil {
			// 			log.Printf("gi.ColorSpec.ParseXML linear gradient parsing error: %v\n", err)
			// 			return false
			// 		}
			// 	}
			// case "radialGradient":
			// 	cs.Gradient = &rasterx.Gradient{Points: [5]float32{0.5, 0.5, 0.5, 0.5, 0.5},
			// 		IsRadial: true, Matrix: rasterx.Identity}
			// 	var setFx, setFy bool
			// 	for _, attr := range se.Attr {
			// 		switch attr.Name.Local {
			// 		case "id":
			// 			// id := attr.Value
			// 			// if len(id) >= 0 {
			// 			// 	svg.Ids[id] = cursor.grad
			// 			// } else {
			// 			// 	return icon, zeroLengthIdError
			// 			// }
			// 		case "r":
			// 			cs.Gradient.Points[4], err = readFraction(attr.Value)
			// 		case "cx":
			// 			cs.Gradient.Points[0], err = readFraction(attr.Value)
			// 		case "cy":
			// 			cs.Gradient.Points[1], err = readFraction(attr.Value)
			// 		case "fx":
			// 			setFx = true
			// 			cs.Gradient.Points[2], err = readFraction(attr.Value)
			// 		case "fy":
			// 			setFy = true
			// 			cs.Gradient.Points[3], err = readFraction(attr.Value)
			// 		default:
			// 			err = cs.ReadGradAttr(attr)
			// 		}
			// 		if err != nil {
			// 			log.Printf("gi.ColorSpec.ParseXML radial gradient parsing error: %v\n", err)
			// 			return false
			// 		}
			// 	}
			// 	if setFx == false { // set fx to cx by default
			// 		cs.Gradient.Points[2] = cs.Gradient.Points[0]
			// 	}
			// 	if setFy == false { // set fy to cy by default
			// 		cs.Gradient.Points[3] = cs.Gradient.Points[1]
			// 	}
			// case "stop":
			// 	stop := rasterx.GradStop{Opacity: 1.0}
			// 	for _, attr := range se.Attr {
			// 		switch attr.Name.Local {
			// 		case "offset":
			// 			stop.Offset, err = readFraction(attr.Value)
			// 		case "stop-color":
			// 			clr, err := ColorFromString(attr.Value, nil)
			// 			if err != nil {
			// 				log.Printf("gi.ColorSpec.ParseXML invalid color string: %v\n", err)
			// 				return false
			// 			}
			// 			stop.StopColor = clr
			// 		case "stop-opacity":
			// 			stop.Opacity, err = strconv.ParseFloat(attr.Value, 64)
			// 		}
			// 		if err != nil {
			// 			log.Printf("gi.ColorSpec.ParseXML color stop parsing error: %v\n", err)
			// 			return false
			// 		}
			// 	}
			// 	cs.Gradient.Stops = append(cs.Gradient.Stops, &stop)
			default:
				errStr := "Cannot process svg element " + se.Name.Local
				log.Println(errStr)
			}
		case xml.EndElement:
			switch se.Name.Local {
			case "title":
				inTitle = false
			case "desc":
				inDesc = false
			case "def":
				if inDef {
				}
				inDef = false
				curPar = defPrevPar
			case "radialGradient", "linearGradient":
				if inGrad {
				}
				inGrad = false
				curPar = curPar.Parent()
			case "svg":
				curPar = curPar.Parent()
				if curPar == svg.This {
					break
				}
				curSvg = curPar.ParentByType(KiT_SVG, true).(*SVG)
			default:
				curPar = curPar.Parent()
			}
		case xml.CharData:
			if inTitle {
				curSvg.Title += string(se)
			}
			if inDesc {
				curSvg.Desc += string(se)
			}
		}
	}
	return nil
}
