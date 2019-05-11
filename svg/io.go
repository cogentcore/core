// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// svg parsing is adapted from github.com/srwiley/oksvg:
//
// Copyright 2017 The oksvg Authors. All rights reserved.
//
// created: 2/12/2017 by S.R.Wiley

package svg

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/units"
	"golang.org/x/net/html/charset"
)

// this file contains all the IO-related parsing etc routines

// see https://github.com/goki/ki/wiki/Naming for IO naming conventions
// using standard XML marshal / unmarshal

var (
	paramMismatchError  = errors.New("gi.SVG Parse: Param mismatch")
	commandUnknownError = errors.New("gi.SVG Parse: Unknown command")
	zeroLengthIdError   = errors.New("gi.SVG Parse: zero length id")
	missingIdError      = errors.New("gi.SVG Parse: cannot find id")
)

// OpenXML Opens XML-formatted SVG input from given file
func (svg *SVG) OpenXML(filename string) error {
	fi, err := os.Stat(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	if fi.IsDir() {
		err := fmt.Errorf("svg.OpenXML: file is a directory: %v\n", filename)
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
// If this is being read into a live scenegraph, then you MUST call
// 	svg.FullInit2DTree() after to initialize it for rendering.
func (svg *SVG) ReadXML(reader io.Reader) error {
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
	decoder.CharsetReader = charset.NewReaderLabel
	var err error
	for {
		var t xml.Token
		t, err = decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("gi.SVG parsing error: %v\n", err)
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			err = svg.UnmarshalXML(decoder, se)
			break
			// todo: ignore rest?
		}
	}
	return err
}

// UnmarshalXML unmarshals the svg using xml.Decoder
func (svg *SVG) UnmarshalXML(decoder *xml.Decoder, se xml.StartElement) error {
	updt := svg.UpdateStart()
	defer svg.UpdateEnd(updt)

	start := &se

	svg.DeleteAll()

	curPar := svg.This().(gi.Node2D) // current parent node into which elements are created
	curSvg := svg
	inTitle := false
	inDesc := false
	inDef := false
	inCSS := false
	var curCSS *gi.StyleSheet
	inTxt := false
	var curTxt *Text
	inTspn := false
	var curTspn *Text
	var defPrevPar gi.Node2D // previous parent before a def encountered

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
				if curPar != svg.This() {
					curPar = curPar.AddNewChild(KiT_SVG, "svg").(gi.Node2D)
				}
				csvg := curPar.Embed(KiT_SVG).(*SVG)
				curSvg = csvg
				for _, attr := range se.Attr {
					if csvg.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "viewBox":
						pts := gi.ReadPoints(attr.Value)
						if len(pts) != 4 {
							return paramMismatchError
						}
						csvg.ViewBox.Min.X = pts[0]
						csvg.ViewBox.Min.Y = pts[1]
						csvg.ViewBox.Size.X = pts[2]
						csvg.ViewBox.Size.Y = pts[3]
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
				curPar = curPar.AddNewChild(KiT_Group, "g").(gi.Node2D)
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
				rect := AddNewRect(curPar, "rect", 0, 0, 1, 1)
				var x, y, w, h, rx, ry float32
				for _, attr := range se.Attr {
					if rect.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x":
						x, err = gi.ParseFloat32(attr.Value)
					case "y":
						y, err = gi.ParseFloat32(attr.Value)
					case "width":
						w, err = gi.ParseFloat32(attr.Value)
					case "height":
						h, err = gi.ParseFloat32(attr.Value)
					case "rx":
						rx, err = gi.ParseFloat32(attr.Value)
					case "ry":
						ry, err = gi.ParseFloat32(attr.Value)
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
				circle := AddNewCircle(curPar, "circle", 0, 0, 1)
				var cx, cy, r float32
				for _, attr := range se.Attr {
					if circle.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "cx":
						cx, err = gi.ParseFloat32(attr.Value)
					case "cy":
						cy, err = gi.ParseFloat32(attr.Value)
					case "r":
						r, err = gi.ParseFloat32(attr.Value)
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
				ellipse := AddNewEllipse(curPar, "ellipse", 0, 0, 1, 1)
				var cx, cy, rx, ry float32
				for _, attr := range se.Attr {
					if ellipse.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "cx":
						cx, err = gi.ParseFloat32(attr.Value)
					case "cy":
						cy, err = gi.ParseFloat32(attr.Value)
					case "rx":
						rx, err = gi.ParseFloat32(attr.Value)
					case "ry":
						ry, err = gi.ParseFloat32(attr.Value)
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
				line := AddNewLine(curPar, "line", 0, 0, 1, 1)
				var x1, x2, y1, y2 float32
				for _, attr := range se.Attr {
					if line.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x1":
						x1, err = gi.ParseFloat32(attr.Value)
					case "y1":
						y1, err = gi.ParseFloat32(attr.Value)
					case "x2":
						x2, err = gi.ParseFloat32(attr.Value)
					case "y2":
						y2, err = gi.ParseFloat32(attr.Value)
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
				polygon := AddNewPolygon(curPar, "polygon", nil)
				for _, attr := range se.Attr {
					if polygon.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "points":
						pts := gi.ReadPoints(attr.Value)
						if pts != nil {
							sz := len(pts)
							if sz%2 != 0 {
								err = fmt.Errorf("gi.SVG polygon has an odd number of points: %v str: %v\n", sz, attr.Value)
								log.Println(err)
								return err
							}
							pvec := make([]gi.Vec2D, sz/2)
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
				polyline := AddNewPolyline(curPar, "polyline", nil)
				for _, attr := range se.Attr {
					if polyline.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "points":
						pts := gi.ReadPoints(attr.Value)
						if pts != nil {
							sz := len(pts)
							if sz%2 != 0 {
								err = fmt.Errorf("gi.SVG polyline has an odd number of points: %v str: %v\n", sz, attr.Value)
								log.Println(err)
								return err
							}
							pvec := make([]gi.Vec2D, sz/2)
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
				path := AddNewPath(curPar, "path", "")
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
				var txt *Text
				if se.Name.Local == "text" {
					txt = AddNewText(curPar, "txt", 0, 0, "")
					inTxt = true
					curTxt = txt
				} else {
					if inTxt && curTxt != nil {
						txt = AddNewText(curTxt, "tspan", 0, 0, "")
						txt.Pos = curTxt.Pos
					} else {
						txt = AddNewText(curPar, "tspan", 0, 0, "")
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
						pts := gi.ReadPoints(attr.Value)
						if len(pts) > 1 {
							txt.CharPosX = pts
						} else if len(pts) == 1 {
							txt.Pos.X = pts[0]
						}
					case "y":
						pts := gi.ReadPoints(attr.Value)
						if len(pts) > 1 {
							txt.CharPosY = pts
						} else if len(pts) == 1 {
							txt.Pos.Y = pts[0]
						}
					case "dx":
						pts := gi.ReadPoints(attr.Value)
						if len(pts) > 0 {
							txt.CharPosDX = pts
						}
					case "dy":
						pts := gi.ReadPoints(attr.Value)
						if len(pts) > 0 {
							txt.CharPosDY = pts
						}
					case "rotate":
						pts := gi.ReadPoints(attr.Value)
						if len(pts) > 0 {
							txt.CharRots = pts
						}
					case "textLength":
						tl, err := gi.ParseFloat32(attr.Value)
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
				grad := gi.AddNewGradient(curPar, "lin-grad")
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
							if hrg, ok := hr.(*gi.Gradient); ok {
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
				grad := gi.AddNewGradient(curPar, "rad-grad")
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
							if hrg, ok := hr.(*gi.Gradient); ok {
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
				sty := gi.AddNewStyleSheet(curPar, "style")
				for _, attr := range se.Attr {
					if sty.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
				}
				inCSS = true
				curCSS = sty
				// style code shows up in CharData below
			case nm == "clipPath":
				curPar = curPar.AddNewChild(KiT_ClipPath, "clip-path").(gi.Node2D)
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
				curPar = curPar.AddNewChild(KiT_Marker, "marker").(gi.Node2D)
				mrk := curPar.(*Marker)
				var rx, ry float32
				szx := float32(3)
				szy := float32(3)
				for _, attr := range se.Attr {
					if mrk.SetStdXMLAttr(attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "refX":
						rx, err = gi.ParseFloat32(attr.Value)
					case "refY":
						ry, err = gi.ParseFloat32(attr.Value)
					case "markerWidth":
						szx, err = gi.ParseFloat32(attr.Value)
					case "markerHeight":
						szy, err = gi.ParseFloat32(attr.Value)
					case "matrixUnits":
						if attr.Value == "strokeWidth" {
							mrk.Units = StrokeWidth
						} else {
							mrk.Units = UserSpaceOnUse
						}
					case "viewBox":
						pts := gi.ReadPoints(attr.Value)
						if len(pts) != 4 {
							return paramMismatchError
						}
						mrk.ViewBox.Min.X = pts[0]
						mrk.ViewBox.Min.Y = pts[1]
						mrk.ViewBox.Size.X = pts[2]
						mrk.ViewBox.Size.Y = pts[3]
					case "orient":
						mrk.Orient = attr.Value
					default:
						mrk.SetProp(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
				mrk.RefPos.Set(rx, ry)
				mrk.Size.Set(szx, szy)
			case nm == "use":
				link := gi.XMLAttr("href", se.Attr)
				itm := curPar.FindNamedElement(link)
				if itm != nil {
					cln := itm.Clone().(gi.Node2D)
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
				curPar = curPar.AddNewChild(gi.KiT_MetaData2D, nm).(gi.Node2D)
				md := curPar.(*gi.MetaData2D)
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
				curPar = curPar.AddNewChild(KiT_Flow, nm).(gi.Node2D)
				md := curPar.(*Flow)
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
				fallthrough
			case strings.HasPrefix(nm, "path-effect"):
				fallthrough
			case strings.HasPrefix(nm, "filter"):
				curPar = curPar.AddNewChild(KiT_Filter, nm).(gi.Node2D)
				md := curPar.(*Filter)
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
				IconAutoOpen = false
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
				if curPar == svg.This() {
					break
				}
				if curPar.Parent() == nil {
					break
				}
				curPar = curPar.Parent().(gi.Node2D)
				if curPar == svg.This() {
					break
				}
				curSvgk := curPar.ParentByType(KiT_SVG, true)
				if curSvgk != nil {
					curSvg = curSvgk.Embed(KiT_SVG).(*SVG)
				}
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
