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
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/srwiley/rasterx"
	"goki.dev/colors"
	"goki.dev/girl/gist"
	"goki.dev/ki/v2"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
	"golang.org/x/net/html/charset"
)

// this file contains all the IO-related parsing etc routines

// see https://goki.dev/ki/v2/wiki/Naming for IO naming conventions
// using standard XML marshal / unmarshal

var (
	paramMismatchError  = errors.New("SVG Parse: Param mismatch")
	commandUnknownError = errors.New("SVG Parse: Unknown command")
	zeroLengthIdError   = errors.New("SVG Parse: zero length id")
	missingIdError      = errors.New("SVG Parse: cannot find id")
)

// OpenXML Opens XML-formatted SVG input from given file
func (sv *SVG) OpenXML(fname string) error {
	filename := string(fname)
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
	return sv.ReadXML(bufio.NewReader(fp))
}

// ReadXML reads XML-formatted SVG input from io.Reader, and uses
// xml.Decoder to create the SVG scenegraph for corresponding SVG drawing.
// Removes any existing content in SVG first. To process a byte slice, pass:
// bytes.NewReader([]byte(str)) -- all errors are logged and also returned.
// If this is being read into a live scenegraph, then you MUST call
//
//	svg.FullInit2DTree() after to initialize it for rendering.
func (sv *SVG) ReadXML(reader io.Reader) error {
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
			log.Printf("SVG parsing error: %v\n", err)
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			err = sv.UnmarshalXML(decoder, se)
			break
			// todo: ignore rest?
		}
	}
	if err == io.EOF {
		return nil
	}
	return err
}

// UnmarshalXML unmarshals the svg using xml.Decoder
func (sv *SVG) UnmarshalXML(decoder *xml.Decoder, se xml.StartElement) error {
	start := &se

	sv.DeleteAll()

	curPar := sv.Root.This().(Node) // current parent node into which elements are created
	curSvg := &sv.Root
	inTitle := false
	inDesc := false
	inDef := false
	inCSS := false
	var curCSS *StyleSheet
	inTxt := false
	var curTxt *Text
	inTspn := false
	var curTspn *Text
	var defPrevPar Node // previous parent before a def encountered

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
			log.Printf("SVG parsing error: %v\n", err)
			return err
		}
		switch se := t.(type) {
		case xml.StartElement:
			nm := se.Name.Local
			switch {
			case nm == "svg":
				// if curPar != sv.This() {
				// 	curPar = curPar.NewChild(TypeSVG, "svg").(Node)
				// }
				for _, attr := range se.Attr {
					// if SetStdXMLAttr(curSvg, attr.Name.Local, attr.Value) {
					// 	continue
					// }
					switch attr.Name.Local {
					case "viewBox":
						pts := mat32.ReadPoints(attr.Value)
						if len(pts) != 4 {
							return paramMismatchError
						}
						curSvg.ViewBox.Min.X = pts[0]
						curSvg.ViewBox.Min.Y = pts[1]
						curSvg.ViewBox.Size.X = pts[2]
						curSvg.ViewBox.Size.Y = pts[3]
					case "width":
						sv.PhysWidth.SetString(attr.Value)
						sv.PhysWidth.ToDots(&curSvg.Paint.UnContext)
					case "height":
						sv.PhysHeight.SetString(attr.Value)
						sv.PhysHeight.ToDots(&curSvg.Paint.UnContext)
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
				curPar = &sv.Defs
			case nm == "g":
				curPar = curPar.NewChild(GroupType, "g").(Node)
				for _, attr := range se.Attr {
					if SetStdXMLAttr(curPar.AsNodeBase(), attr.Name.Local, attr.Value) {
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
					if SetStdXMLAttr(rect, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x":
						x, err = mat32.ParseFloat32(attr.Value)
					case "y":
						y, err = mat32.ParseFloat32(attr.Value)
					case "width":
						w, err = mat32.ParseFloat32(attr.Value)
					case "height":
						h, err = mat32.ParseFloat32(attr.Value)
					case "rx":
						rx, err = mat32.ParseFloat32(attr.Value)
					case "ry":
						ry, err = mat32.ParseFloat32(attr.Value)
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
					if SetStdXMLAttr(circle, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "cx":
						cx, err = mat32.ParseFloat32(attr.Value)
					case "cy":
						cy, err = mat32.ParseFloat32(attr.Value)
					case "r":
						r, err = mat32.ParseFloat32(attr.Value)
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
					if SetStdXMLAttr(ellipse, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "cx":
						cx, err = mat32.ParseFloat32(attr.Value)
					case "cy":
						cy, err = mat32.ParseFloat32(attr.Value)
					case "rx":
						rx, err = mat32.ParseFloat32(attr.Value)
					case "ry":
						ry, err = mat32.ParseFloat32(attr.Value)
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
					if SetStdXMLAttr(line, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x1":
						x1, err = mat32.ParseFloat32(attr.Value)
					case "y1":
						y1, err = mat32.ParseFloat32(attr.Value)
					case "x2":
						x2, err = mat32.ParseFloat32(attr.Value)
					case "y2":
						y2, err = mat32.ParseFloat32(attr.Value)
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
					if SetStdXMLAttr(polygon, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "points":
						pts := mat32.ReadPoints(attr.Value)
						if pts != nil {
							sz := len(pts)
							if sz%2 != 0 {
								err = fmt.Errorf("SVG polygon has an odd number of points: %v str: %v\n", sz, attr.Value)
								log.Println(err)
								return err
							}
							pvec := make([]mat32.Vec2, sz/2)
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
					if SetStdXMLAttr(polyline, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "points":
						pts := mat32.ReadPoints(attr.Value)
						if pts != nil {
							sz := len(pts)
							if sz%2 != 0 {
								err = fmt.Errorf("SVG polyline has an odd number of points: %v str: %v\n", sz, attr.Value)
								log.Println(err)
								return err
							}
							pvec := make([]mat32.Vec2, sz/2)
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
					if attr.Name.Local == "original-d" {
						continue
					}
					if SetStdXMLAttr(path, attr.Name.Local, attr.Value) {
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
			case nm == "image":
				img := AddNewImage(curPar, "image", 0, 0)
				var x, y, w, h float32
				b := false
				for _, attr := range se.Attr {
					if SetStdXMLAttr(img, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x":
						x, err = mat32.ParseFloat32(attr.Value)
					case "y":
						y, err = mat32.ParseFloat32(attr.Value)
					case "width":
						w, err = mat32.ParseFloat32(attr.Value)
					case "height":
						h, err = mat32.ParseFloat32(attr.Value)
					case "preserveAspectRatio":
						b, _ = laser.ToBool(attr.Value)
						img.PreserveAspectRatio = b
					case "href":
						if len(attr.Value) > 11 && attr.Value[:11] == "data:image/" {
							es := attr.Value[11:]
							fmti := strings.Index(es, ";")
							fm := es[:fmti]
							bs64 := es[fmti+1 : fmti+8]
							if bs64 != "base64," {
								log.Printf("image base64 encoding string not properly formatted: %s\n", bs64)
							}
							eb := []byte(es[fmti+8:])
							im, err := ImageFmBase64(fm, eb)
							if err != nil {
								log.Println(err)
							} else {
								img.SetImage(im, 0, 0)
							}
						} else { // url

						}
					default:
						img.SetProp(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
				img.Pos.Set(x, y)
				img.Size.Set(w, h)
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
					if SetStdXMLAttr(txt, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x":
						pts := mat32.ReadPoints(attr.Value)
						if len(pts) > 1 {
							txt.CharPosX = pts
						} else if len(pts) == 1 {
							txt.Pos.X = pts[0]
						}
					case "y":
						pts := mat32.ReadPoints(attr.Value)
						if len(pts) > 1 {
							txt.CharPosY = pts
						} else if len(pts) == 1 {
							txt.Pos.Y = pts[0]
						}
					case "dx":
						pts := mat32.ReadPoints(attr.Value)
						if len(pts) > 0 {
							txt.CharPosDX = pts
						}
					case "dy":
						pts := mat32.ReadPoints(attr.Value)
						if len(pts) > 0 {
							txt.CharPosDY = pts
						}
					case "rotate":
						pts := mat32.ReadPoints(attr.Value)
						if len(pts) > 0 {
							txt.CharRots = pts
						}
					case "textLength":
						tl, err := mat32.ParseFloat32(attr.Value)
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
				grad := NewGradient(curPar, "lin-grad")
				for _, attr := range se.Attr {
					if SetStdXMLAttr(grad, attr.Name.Local, attr.Value) {
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
								grad.StopsName = nm
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
				grad := NewGradient(curPar, "rad-grad")
				for _, attr := range se.Attr {
					if SetStdXMLAttr(grad, attr.Name.Local, attr.Value) {
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
								grad.StopsName = nm
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
				sty := NewStyleSheet(curPar, "style")
				for _, attr := range se.Attr {
					if SetStdXMLAttr(sty, attr.Name.Local, attr.Value) {
						continue
					}
				}
				inCSS = true
				curCSS = sty
				// style code shows up in CharData below
			case nm == "clipPath":
				curPar = curPar.NewChild(ClipPathType, "clip-path").(Node)
				cp := curPar.(*ClipPath)
				for _, attr := range se.Attr {
					if SetStdXMLAttr(cp, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						cp.SetProp(attr.Name.Local, attr.Value)
					}
				}
			case nm == "marker":
				curPar = curPar.NewChild(MarkerType, "marker").(Node)
				mrk := curPar.(*Marker)
				var rx, ry float32
				szx := float32(3)
				szy := float32(3)
				for _, attr := range se.Attr {
					if SetStdXMLAttr(mrk, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "refX":
						rx, err = mat32.ParseFloat32(attr.Value)
					case "refY":
						ry, err = mat32.ParseFloat32(attr.Value)
					case "markerWidth":
						szx, err = mat32.ParseFloat32(attr.Value)
					case "markerHeight":
						szy, err = mat32.ParseFloat32(attr.Value)
					case "matrixUnits":
						if attr.Value == "strokeWidth" {
							mrk.Units = StrokeWidth
						} else {
							mrk.Units = UserSpaceOnUse
						}
					case "viewBox":
						pts := mat32.ReadPoints(attr.Value)
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
				link := gist.XMLAttr("href", se.Attr)
				itm := sv.FindNamedElement(link)
				if itm != nil {
					cln := itm.Clone().(Node)
					if cln != nil {
						curPar.AddChild(cln)
						for _, attr := range se.Attr {
							if SetStdXMLAttr(cln.AsNodeBase(), attr.Name.Local, attr.Value) {
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
				curPar = curPar.NewChild(MetaDataType, nm).(Node)
				md := curPar.(*MetaData)
				md.Class = nm
				for _, attr := range se.Attr {
					if SetStdXMLAttr(md, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						curPar.SetProp(attr.Name.Local, attr.Value)
					}
				}
			case strings.HasPrefix(nm, "flow"):
				curPar = curPar.NewChild(FlowType, nm).(Node)
				md := curPar.(*Flow)
				md.Class = nm
				md.FlowType = nm
				for _, attr := range se.Attr {
					if SetStdXMLAttr(md, attr.Name.Local, attr.Value) {
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
				curPar = curPar.NewChild(FilterType, nm).(Node)
				md := curPar.(*Filter)
				md.Class = nm
				md.FilterType = nm
				for _, attr := range se.Attr {
					if SetStdXMLAttr(md, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						curPar.SetProp(attr.Name.Local, attr.Value)
					}
				}
			default:
				errStr := "SVG Cannot process svg element " + se.Name.Local
				log.Println(errStr)
				// IconAutoOpen = false
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
				if curPar == sv.Root.This() {
					break
				}
				if curPar.Parent() == nil {
					break
				}
				curPar = curPar.Parent().(Node)
				if curPar == sv.Root.This() {
					break
				}
				curSvgk := curPar.ParentByType(SVGNodeType, ki.NoEmbeds)
				if curSvgk != nil {
					curSvg = curSvgk.(*SVGNode)
				}
			}
		case xml.CharData:
			// (ok, md := curPar.(*MetaData); ok)
			trspc := strings.TrimSpace(string(se))
			switch {
			// case :
			// 	md.MetaData = string(se)
			case inTitle:
				sv.Title += trspc
			case inDesc:
				sv.Desc += trspc
			case inTspn && curTspn != nil:
				curTspn.Text = trspc
			case inTxt && curTxt != nil:
				curTxt.Text = trspc
			case inCSS && curCSS != nil:
				curCSS.ParseString(trspc)
				cp := curCSS.CSSProps()
				if cp != nil {
					if inDef && defPrevPar != nil {
						defPrevPar.AsNodeBase().CSS = cp
					} else {
						curPar.AsNodeBase().CSS = cp
					}
				}
			}
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////////
//   Writing

// SaveXML saves the svg to a XML-encoded file, using WriteXML
func (sv *SVG) SaveXML(fname string) error {
	filename := string(fname)
	fp, err := os.Create(filename)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	bw := bufio.NewWriter(fp)
	err = sv.WriteXML(bw, true)
	if err != nil {
		log.Println(err)
		return err
	}
	err = bw.Flush()
	if err != nil {
		log.Println(err)
	}
	return err
}

// WriteXML writes XML-formatted SVG output to io.Writer, and uses
// XMLEncoder
func (sv *SVG) WriteXML(wr io.Writer, indent bool) error {
	enc := NewXMLEncoder(wr)
	if indent {
		enc.Indent("", "  ")
	}
	sv.MarshalXMLx(enc, xml.StartElement{})
	enc.Flush()
	return nil
}

func XMLAddAttr(attr *[]xml.Attr, name, val string) {
	at := xml.Attr{}
	at.Name.Local = name
	at.Value = val
	*attr = append(*attr, at)
}

// InkscapeProps are property keys that should be prefixed with "inkscape:"
var InkscapeProps = map[string]bool{
	"isstock": true,
	"stockid": true,
}

// SVGNodeMarshalXML encodes just the given node under SVG to XML.
// returns name of node, for end tag -- if empty, then children will not be
// output.
func SVGNodeMarshalXML(itm ki.Ki, enc *XMLEncoder, setName string) string {
	if itm == nil || itm.This() == nil {
		return ""
	}
	se := xml.StartElement{}
	var props ki.Props
	if itm.Properties() != nil {
		props = *itm.Properties()
	}
	if itm.Name() != "" {
		XMLAddAttr(&se.Attr, "id", itm.Name())
	}
	text := "" // if non-empty, contains text to render
	_, issvg := itm.(Node)
	_, isgp := itm.(*Group)
	_, ismark := itm.(*Marker)
	if !isgp {
		if issvg && !ismark {
			sp := gist.StylePropsXML(props)
			if sp != "" {
				XMLAddAttr(&se.Attr, "style", sp)
			}
			if txp, has := props["transform"]; has {
				XMLAddAttr(&se.Attr, "transform", laser.ToString(txp))
			}
		} else {
			for k, v := range props {
				sv := laser.ToString(v)
				if _, has := InkscapeProps[k]; has {
					k = "inkscape:" + k
				} else if k == "overflow" {
					k = "style"
					sv = "overflow:" + sv
				}
				XMLAddAttr(&se.Attr, k, sv)
			}
		}
	}
	var sb strings.Builder
	nm := ""
	switch nd := itm.(type) {
	case *Path:
		nm = "path"
		nd.DataStr = PathDataString(nd.Data)
		XMLAddAttr(&se.Attr, "d", nd.DataStr)
	case *Group:
		nm = "g"
		if strings.HasPrefix(strings.ToLower(itm.Name()), "layer") {
		}
		for k, v := range props {
			sv := laser.ToString(v)
			switch k {
			case "opacity", "transform":
				XMLAddAttr(&se.Attr, k, sv)
			case "groupmode":
				XMLAddAttr(&se.Attr, "inkscape:groupmode", sv)
				if st, has := props["style"]; has {
					XMLAddAttr(&se.Attr, "style", laser.ToString(st))
				} else {
					XMLAddAttr(&se.Attr, "style", "display:inline")
				}
			case "insensitive":
				if sv == "true" {
					XMLAddAttr(&se.Attr, "sodipodi:"+k, sv)
				}
			}
		}
	case *Rect:
		nm = "rect"
		XMLAddAttr(&se.Attr, "x", fmt.Sprintf("%g", nd.Pos.X))
		XMLAddAttr(&se.Attr, "y", fmt.Sprintf("%g", nd.Pos.Y))
		XMLAddAttr(&se.Attr, "width", fmt.Sprintf("%g", nd.Size.X))
		XMLAddAttr(&se.Attr, "height", fmt.Sprintf("%g", nd.Size.Y))
	case *Circle:
		nm = "circle"
		XMLAddAttr(&se.Attr, "cx", fmt.Sprintf("%g", nd.Pos.X))
		XMLAddAttr(&se.Attr, "cy", fmt.Sprintf("%g", nd.Pos.Y))
		XMLAddAttr(&se.Attr, "radius", fmt.Sprintf("%g", nd.Radius))
	case *Ellipse:
		nm = "ellipse"
		XMLAddAttr(&se.Attr, "cx", fmt.Sprintf("%g", nd.Pos.X))
		XMLAddAttr(&se.Attr, "cy", fmt.Sprintf("%g", nd.Pos.Y))
		XMLAddAttr(&se.Attr, "rx", fmt.Sprintf("%g", nd.Radii.X))
		XMLAddAttr(&se.Attr, "ry", fmt.Sprintf("%g", nd.Radii.Y))
	case *Line:
		nm = "line"
		XMLAddAttr(&se.Attr, "x1", fmt.Sprintf("%g", nd.Start.X))
		XMLAddAttr(&se.Attr, "y1", fmt.Sprintf("%g", nd.Start.Y))
		XMLAddAttr(&se.Attr, "x2", fmt.Sprintf("%g", nd.End.X))
		XMLAddAttr(&se.Attr, "y2", fmt.Sprintf("%g", nd.End.Y))
	case *Polygon:
		nm = "polygon"
		for _, p := range nd.Points {
			sb.WriteString(fmt.Sprintf("%g,%g ", p.X, p.Y))
		}
		XMLAddAttr(&se.Attr, "points", sb.String())
	case *Polyline:
		nm = "polyline"
		for _, p := range nd.Points {
			sb.WriteString(fmt.Sprintf("%g,%g ", p.X, p.Y))
		}
		XMLAddAttr(&se.Attr, "points", sb.String())
	case *Text:
		if nd.Text == "" {
			nm = "text"
		} else {
			nm = "tspan"
		}
		XMLAddAttr(&se.Attr, "x", fmt.Sprintf("%g", nd.Pos.X))
		XMLAddAttr(&se.Attr, "y", fmt.Sprintf("%g", nd.Pos.Y))
		text = nd.Text
	case *Image:
		if nd.Pixels == nil {
			return ""
		}
		nm = "image"
		XMLAddAttr(&se.Attr, "x", fmt.Sprintf("%g", nd.Pos.X))
		XMLAddAttr(&se.Attr, "y", fmt.Sprintf("%g", nd.Pos.Y))
		XMLAddAttr(&se.Attr, "width", fmt.Sprintf("%g", nd.Size.X))
		XMLAddAttr(&se.Attr, "height", fmt.Sprintf("%g", nd.Size.Y))
		XMLAddAttr(&se.Attr, "preserveAspectRatio", fmt.Sprintf("%v", nd.PreserveAspectRatio))
		ib, fmt := ImageToBase64PNG(nd.Pixels)
		XMLAddAttr(&se.Attr, "xlink:href", "data:"+fmt+";base64,"+string(Base64SplitLines(ib)))
	case *MetaData:
		if strings.HasPrefix(nd.Nm, "namedview") {
			nm = "sodipodi:namedview"
		} else if strings.HasPrefix(nd.Nm, "grid") {
			nm = "inkscape:grid"
		}
	case *Gradient:
		SVGNodeXMLGrad(nd, nd.Nm, enc)
		return "" // exclude -- already written
	case *Marker:
		nm = "marker"
		XMLAddAttr(&se.Attr, "refX", fmt.Sprintf("%g", nd.RefPos.X))
		XMLAddAttr(&se.Attr, "refY", fmt.Sprintf("%g", nd.RefPos.Y))
		XMLAddAttr(&se.Attr, "orient", nd.Orient)
	case *Filter:
		return "" // not yet supported
	case *StyleSheet:
		nm = "style"
	default:
		nm = itm.Type().Name
	}
	se.Name.Local = nm
	if setName != "" {
		se.Name.Local = setName
	}
	enc.EncodeToken(se)
	if text != "" {
		cd := xml.CharData([]byte(text))
		enc.EncodeToken(cd)
	}
	return se.Name.Local
}

func SVGNodeXMLGrad(nd *Gradient, name string, enc *XMLEncoder) {
	cs := &nd.Grad
	if cs.Gradient == nil {
		return
	}
	gr := cs.Gradient
	me := xml.StartElement{}
	XMLAddAttr(&me.Attr, "id", name)

	linear := true
	if cs.Source == gist.LinearGradient {
		me.Name.Local = "linearGradient"
	} else {
		linear = false
		me.Name.Local = "radialGradient"
	}

	nilpts := gr.Points[0] == 0 && gr.Points[1] == 0 && gr.Points[2] == 1 && gr.Points[3] == 0 && gr.Points[4] == 0
	if !nilpts {
		if linear {
			XMLAddAttr(&me.Attr, "x1", fmt.Sprintf("%g", gr.Points[0]))
			XMLAddAttr(&me.Attr, "y1", fmt.Sprintf("%g", gr.Points[1]))
			XMLAddAttr(&me.Attr, "x2", fmt.Sprintf("%g", gr.Points[2]))
			XMLAddAttr(&me.Attr, "y2", fmt.Sprintf("%g", gr.Points[3]))
		} else {
			XMLAddAttr(&me.Attr, "cx", fmt.Sprintf("%g", gr.Points[0]))
			XMLAddAttr(&me.Attr, "cy", fmt.Sprintf("%g", gr.Points[1]))
			XMLAddAttr(&me.Attr, "fx", fmt.Sprintf("%g", gr.Points[2]))
			XMLAddAttr(&me.Attr, "fy", fmt.Sprintf("%g", gr.Points[3]))
			XMLAddAttr(&me.Attr, "r", fmt.Sprintf("%g", gr.Points[4]))
		}
		if gr.Units == rasterx.ObjectBoundingBox {
			XMLAddAttr(&me.Attr, "gradientUnits", "objectBoundingBox")
		} else {
			XMLAddAttr(&me.Attr, "gradientUnits", "userSpaceOnUse")
		}
	}
	switch gr.Spread {
	case rasterx.ReflectSpread:
		XMLAddAttr(&me.Attr, "spreadMethod", "reflect")
	case rasterx.RepeatSpread:
		XMLAddAttr(&me.Attr, "spreadMethod", "repeat")
	}

	idxf := gr.Matrix == rasterx.Identity
	if !idxf {
		XMLAddAttr(&me.Attr, "gradientTransform", fmt.Sprintf("matrix(%g,%g,%g,%g,%g,%g)", gr.Matrix.A, gr.Matrix.B, gr.Matrix.C, gr.Matrix.D, gr.Matrix.E, gr.Matrix.F))
	}

	if nd.StopsName != "" {
		XMLAddAttr(&me.Attr, "xlink:href", "#"+nd.StopsName)
	}

	enc.EncodeToken(me)
	if nd.StopsName == "" {
		for _, gs := range gr.Stops {
			se := xml.StartElement{}
			se.Name.Local = "stop"
			clr := colors.AsRGBA(gs.StopColor)
			hs := colors.AsHex(clr)[:7] // get rid of transparency
			XMLAddAttr(&se.Attr, "style", fmt.Sprintf("stop-color:%s;stop-opacity:%g;", hs, gs.Opacity))
			XMLAddAttr(&se.Attr, "offset", fmt.Sprintf("%g", gs.Offset))
			enc.EncodeToken(se)
			enc.WriteEnd(se.Name.Local)
		}
	}
	enc.WriteEnd(me.Name.Local)
}

// SVGNodeTreeMarshalXML encodes item and any children to XML.
// returns any error, and name of element that enc.WriteEnd() should be
// called with -- allows for extra elements to be added at end of list.
func SVGNodeTreeMarshalXML(itm Node, enc *XMLEncoder, setName string) (string, error) {
	name := SVGNodeMarshalXML(itm, enc, setName)
	if name == "" {
		return "", nil
	}
	for _, k := range *itm.Children() {
		knm, err := SVGNodeTreeMarshalXML(k.(Node), enc, "")
		if knm != "" {
			enc.WriteEnd(knm)
		}
		if err != nil {
			return name, err
		}
	}
	return name, nil
}

// MarshalXMLx marshals the svg using XMLEncoder
func (sv *SVG) MarshalXMLx(enc *XMLEncoder, se xml.StartElement) error {
	me := xml.StartElement{}
	me.Name.Local = "svg"
	// todo: look for props about units?
	XMLAddAttr(&me.Attr, "width", sv.PhysWidth.String())
	XMLAddAttr(&me.Attr, "height", sv.PhysHeight.String())
	XMLAddAttr(&me.Attr, "viewBox", fmt.Sprintf("%g %g %g %g", sv.Root.ViewBox.Min.X, sv.Root.ViewBox.Min.Y, sv.Root.ViewBox.Size.X, sv.Root.ViewBox.Size.Y))
	XMLAddAttr(&me.Attr, "xmlns:inkscape", "http://www.inkscape.org/namespaces/inkscape")
	XMLAddAttr(&me.Attr, "xmlns:sodipodi", "http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd")
	XMLAddAttr(&me.Attr, "xmlns:xlink", "http://www.w3.org/1999/xlink")
	XMLAddAttr(&me.Attr, "xmlns", "http://www.w3.org/2000/svg")
	enc.EncodeToken(me)

	dnm, err := SVGNodeTreeMarshalXML(&sv.Defs, enc, "defs")
	enc.WriteEnd(dnm)

	for _, k := range sv.Root.Kids {
		var knm string
		knm, err = SVGNodeTreeMarshalXML(k.(Node), enc, "")
		if knm != "" {
			enc.WriteEnd(knm)
		}
		if err != nil {
			break
		}
	}

	ed := xml.EndElement{}
	ed.Name = me.Name
	enc.EncodeToken(ed)
	return err
}

// SetStdXMLAttr sets standard attributes of node given XML-style name /
// attribute values (e.g., from parsing XML / SVG files) -- returns true if handled
func SetStdXMLAttr(ni Node, name, val string) bool {
	nb := ni.AsNodeBase()
	switch name {
	case "id":
		nb.SetName(val)
		return true
	case "class":
		nb.Class = val
		return true
	case "style":
		gist.SetStylePropsXML(val, (*map[string]any)(&nb.Props))
		return true
	}
	return false
}
