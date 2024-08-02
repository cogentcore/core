// Copyright (c) 2018, Cogent Core. All rights reserved.
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
	"io/fs"
	"log"
	"os"
	"strings"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"golang.org/x/net/html/charset"
)

// this file contains all the IO-related parsing etc routines

// see https://cogentcore.org/core/ki/wiki/Naming for IO naming conventions
// using standard XML marshal / unmarshal

var (
	errParamMismatch  = errors.New("SVG Parse: Param mismatch")
	errCommandUnknown = errors.New("SVG Parse: Unknown command")
	errZeroLengthID   = errors.New("SVG Parse: zero length id")
	errMissingID      = errors.New("SVG Parse: cannot find id")
)

// OpenXML Opens XML-formatted SVG input from given file
func (sv *SVG) OpenXML(fname string) error {
	filename := fname
	fi, err := os.Stat(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	if fi.IsDir() {
		err := fmt.Errorf("svg.OpenXML: file is a directory: %v", filename)
		log.Println(err)
		return err
	}
	fp, err := os.Open(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	defer fp.Close()
	return sv.ReadXML(bufio.NewReader(fp))
}

// OpenFS Opens XML-formatted SVG input from given file, filesystem FS
func (sv *SVG) OpenFS(fsys fs.FS, fname string) error {
	fp, err := fsys.Open(fname)
	if err != nil {
		return err
	}
	defer fp.Close()
	return sv.ReadXML(bufio.NewReader(fp))
}

// ReadXML reads XML-formatted SVG input from io.Reader, and uses
// xml.Decoder to create the SVG scenegraph for corresponding SVG drawing.
// Removes any existing content in SVG first. To process a byte slice, pass:
// bytes.NewReader([]byte(str)) -- all errors are logged and also returned.
func (sv *SVG) ReadXML(reader io.Reader) error {
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
	decoder.CharsetReader = charset.NewReaderLabel
	var err error
outer:
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
			break outer
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

	curPar := sv.Root.This.(Node) // current parent node into which elements are created
	curSvg := sv.Root
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
				// if curPar != sv.This {
				// 	curPar = curPar.NewChild(TypeSVG, "svg").(Node)
				// }
				for _, attr := range se.Attr {
					// if SetStdXMLAttr(curSvg, attr.Name.Local, attr.Value) {
					// 	continue
					// }
					switch attr.Name.Local {
					case "viewBox":
						pts := math32.ReadPoints(attr.Value)
						if len(pts) != 4 {
							return errParamMismatch
						}
						curSvg.ViewBox.Min.X = pts[0]
						curSvg.ViewBox.Min.Y = pts[1]
						curSvg.ViewBox.Size.X = pts[2]
						curSvg.ViewBox.Size.Y = pts[3]
					case "width":
						sv.PhysicalWidth.SetString(attr.Value)
						sv.PhysicalWidth.ToDots(&curSvg.Paint.UnitContext)
					case "height":
						sv.PhysicalHeight.SetString(attr.Value)
						sv.PhysicalHeight.ToDots(&curSvg.Paint.UnitContext)
					case "preserveAspectRatio":
						curSvg.ViewBox.PreserveAspectRatio.SetString(attr.Value)
					default:
						curPar.AsTree().SetProperty(attr.Name.Local, attr.Value)
					}
				}
			case nm == "desc":
				inDesc = true
			case nm == "title":
				inTitle = true
			case nm == "defs":
				inDef = true
				defPrevPar = curPar
				curPar = sv.Defs
			case nm == "g":
				curPar = NewGroup(curPar)
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(curPar.AsNodeBase(), attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						curPar.AsTree().SetProperty(attr.Name.Local, attr.Value)
					}
				}
			case nm == "rect":
				rect := NewRect(curPar)
				var x, y, w, h, rx, ry float32
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(rect, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x":
						x, err = math32.ParseFloat32(attr.Value)
					case "y":
						y, err = math32.ParseFloat32(attr.Value)
					case "width":
						w, err = math32.ParseFloat32(attr.Value)
					case "height":
						h, err = math32.ParseFloat32(attr.Value)
					case "rx":
						rx, err = math32.ParseFloat32(attr.Value)
					case "ry":
						ry, err = math32.ParseFloat32(attr.Value)
					default:
						rect.SetProperty(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
				rect.Pos.Set(x, y)
				rect.Size.Set(w, h)
				rect.Radius.Set(rx, ry)
			case nm == "circle":
				circle := NewCircle(curPar)
				var cx, cy, r float32
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(circle, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "cx":
						cx, err = math32.ParseFloat32(attr.Value)
					case "cy":
						cy, err = math32.ParseFloat32(attr.Value)
					case "r":
						r, err = math32.ParseFloat32(attr.Value)
					default:
						circle.SetProperty(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
				circle.Pos.Set(cx, cy)
				circle.Radius = r
			case nm == "ellipse":
				ellipse := NewEllipse(curPar)
				var cx, cy, rx, ry float32
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(ellipse, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "cx":
						cx, err = math32.ParseFloat32(attr.Value)
					case "cy":
						cy, err = math32.ParseFloat32(attr.Value)
					case "rx":
						rx, err = math32.ParseFloat32(attr.Value)
					case "ry":
						ry, err = math32.ParseFloat32(attr.Value)
					default:
						ellipse.SetProperty(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
				ellipse.Pos.Set(cx, cy)
				ellipse.Radii.Set(rx, ry)
			case nm == "line":
				line := NewLine(curPar)
				var x1, x2, y1, y2 float32
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(line, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x1":
						x1, err = math32.ParseFloat32(attr.Value)
					case "y1":
						y1, err = math32.ParseFloat32(attr.Value)
					case "x2":
						x2, err = math32.ParseFloat32(attr.Value)
					case "y2":
						y2, err = math32.ParseFloat32(attr.Value)
					default:
						line.SetProperty(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
				line.Start.Set(x1, y1)
				line.End.Set(x2, y2)
			case nm == "polygon":
				polygon := NewPolygon(curPar)
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(polygon, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "points":
						pts := math32.ReadPoints(attr.Value)
						if pts != nil {
							sz := len(pts)
							if sz%2 != 0 {
								err = fmt.Errorf("SVG polygon has an odd number of points: %v str: %v", sz, attr.Value)
								log.Println(err)
								return err
							}
							pvec := make([]math32.Vector2, sz/2)
							for ci := 0; ci < sz/2; ci++ {
								pvec[ci].Set(pts[ci*2], pts[ci*2+1])
							}
							polygon.Points = pvec
						}
					default:
						polygon.SetProperty(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
			case nm == "polyline":
				polyline := NewPolyline(curPar)
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(polyline, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "points":
						pts := math32.ReadPoints(attr.Value)
						if pts != nil {
							sz := len(pts)
							if sz%2 != 0 {
								err = fmt.Errorf("SVG polyline has an odd number of points: %v str: %v", sz, attr.Value)
								log.Println(err)
								return err
							}
							pvec := make([]math32.Vector2, sz/2)
							for ci := 0; ci < sz/2; ci++ {
								pvec[ci].Set(pts[ci*2], pts[ci*2+1])
							}
							polyline.Points = pvec
						}
					default:
						polyline.SetProperty(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
			case nm == "path":
				path := NewPath(curPar)
				for _, attr := range se.Attr {
					if attr.Name.Local == "original-d" {
						continue
					}
					if SetStandardXMLAttr(path, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "d":
						path.SetData(attr.Value)
					default:
						path.SetProperty(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
			case nm == "image":
				img := NewImage(curPar)
				var x, y, w, h float32
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(img, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x":
						x, err = math32.ParseFloat32(attr.Value)
					case "y":
						y, err = math32.ParseFloat32(attr.Value)
					case "width":
						w, err = math32.ParseFloat32(attr.Value)
					case "height":
						h, err = math32.ParseFloat32(attr.Value)
					case "preserveAspectRatio":
						img.ViewBox.PreserveAspectRatio.SetString(attr.Value)
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
							im, err := imagex.FromBase64(fm, eb)
							if err != nil {
								log.Println(err)
							} else {
								img.SetImage(im, 0, 0)
							}
						} else { // url

						}
					default:
						img.SetProperty(attr.Name.Local, attr.Value)
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
					txt = NewText(curPar)
					inTxt = true
					curTxt = txt
				} else {
					if (inTxt && curTxt != nil) || curPar == nil {
						txt = NewText(curTxt)
						tree.SetUniqueName(txt)
						txt.Pos = curTxt.Pos
					} else if curTxt != nil {
						txt = NewText(curPar)
						tree.SetUniqueName(txt)
					}
					inTspn = true
					curTspn = txt
				}
				if txt == nil {
					break
				}
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(txt, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "x":
						pts := math32.ReadPoints(attr.Value)
						if len(pts) > 1 {
							txt.CharPosX = pts
						} else if len(pts) == 1 {
							txt.Pos.X = pts[0]
						}
					case "y":
						pts := math32.ReadPoints(attr.Value)
						if len(pts) > 1 {
							txt.CharPosY = pts
						} else if len(pts) == 1 {
							txt.Pos.Y = pts[0]
						}
					case "dx":
						pts := math32.ReadPoints(attr.Value)
						if len(pts) > 0 {
							txt.CharPosDX = pts
						}
					case "dy":
						pts := math32.ReadPoints(attr.Value)
						if len(pts) > 0 {
							txt.CharPosDY = pts
						}
					case "rotate":
						pts := math32.ReadPoints(attr.Value)
						if len(pts) > 0 {
							txt.CharRots = pts
						}
					case "textLength":
						tl, err := math32.ParseFloat32(attr.Value)
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
						txt.SetProperty(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
			case nm == "linearGradient":
				grad := NewGradient(curPar)
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(grad, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "href":
						nm := attr.Value
						nm = strings.TrimPrefix(nm, "#")
						hr := curPar.AsTree().ChildByName(nm, 0)
						if hr != nil {
							if hrg, ok := hr.(*Gradient); ok {
								grad.StopsName = nm
								grad.Grad = gradient.CopyOf(hrg.Grad)
								if _, ok := grad.Grad.(*gradient.Linear); !ok {
									cp := grad.Grad
									grad.Grad = gradient.NewLinear()
									*grad.Grad.AsBase() = *cp.AsBase()
								}
							}
						}
					}
				}
				err = gradient.UnmarshalXML(&grad.Grad, decoder, se)
				if err != nil {
					return err
				}
			case nm == "radialGradient":
				grad := NewGradient(curPar)
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(grad, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "href":
						nm := attr.Value
						nm = strings.TrimPrefix(nm, "#")
						hr := curPar.AsTree().ChildByName(nm, 0)
						if hr != nil {
							if hrg, ok := hr.(*Gradient); ok {
								grad.StopsName = nm
								grad.Grad = gradient.CopyOf(hrg.Grad)
								if _, ok := grad.Grad.(*gradient.Radial); !ok {
									cp := grad.Grad
									grad.Grad = gradient.NewRadial()
									*grad.Grad.AsBase() = *cp.AsBase()
								}
							}
						}
					}
				}
				err = gradient.UnmarshalXML(&grad.Grad, decoder, se)
				if err != nil {
					return err
				}
			case nm == "style":
				sty := NewStyleSheet(curPar)
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(sty, attr.Name.Local, attr.Value) {
						continue
					}
				}
				inCSS = true
				curCSS = sty
				// style code shows up in CharData below
			case nm == "clipPath":
				curPar = NewClipPath(curPar)
				cp := curPar.(*ClipPath)
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(cp, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						cp.SetProperty(attr.Name.Local, attr.Value)
					}
				}
			case nm == "marker":
				curPar = NewMarker(curPar)
				mrk := curPar.(*Marker)
				var rx, ry float32
				szx := float32(3)
				szy := float32(3)
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(mrk, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					case "refX":
						rx, err = math32.ParseFloat32(attr.Value)
					case "refY":
						ry, err = math32.ParseFloat32(attr.Value)
					case "markerWidth":
						szx, err = math32.ParseFloat32(attr.Value)
					case "markerHeight":
						szy, err = math32.ParseFloat32(attr.Value)
					case "matrixUnits":
						if attr.Value == "strokeWidth" {
							mrk.Units = StrokeWidth
						} else {
							mrk.Units = UserSpaceOnUse
						}
					case "viewBox":
						pts := math32.ReadPoints(attr.Value)
						if len(pts) != 4 {
							return errParamMismatch
						}
						mrk.ViewBox.Min.X = pts[0]
						mrk.ViewBox.Min.Y = pts[1]
						mrk.ViewBox.Size.X = pts[2]
						mrk.ViewBox.Size.Y = pts[3]
					case "orient":
						mrk.Orient = attr.Value
					default:
						mrk.SetProperty(attr.Name.Local, attr.Value)
					}
					if err != nil {
						return err
					}
				}
				mrk.RefPos.Set(rx, ry)
				mrk.Size.Set(szx, szy)
			case nm == "use":
				link := gradient.XMLAttr("href", se.Attr)
				itm := sv.FindNamedElement(link)
				if itm != nil {
					cln := itm.AsTree().Clone().(Node)
					if cln != nil {
						curPar.AsTree().AddChild(cln)
						for _, attr := range se.Attr {
							if SetStandardXMLAttr(cln.AsNodeBase(), attr.Name.Local, attr.Value) {
								continue
							}
							switch attr.Name.Local {
							default:
								cln.AsTree().SetProperty(attr.Name.Local, attr.Value)
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
				curPar = NewMetaData(curPar)
				md := curPar.(*MetaData)
				md.Class = nm
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(md, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						curPar.AsTree().SetProperty(attr.Name.Local, attr.Value)
					}
				}
			case strings.HasPrefix(nm, "flow"):
				curPar = NewFlow(curPar)
				md := curPar.(*Flow)
				md.Class = nm
				md.FlowType = nm
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(md, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						curPar.AsTree().SetProperty(attr.Name.Local, attr.Value)
					}
				}
			case strings.HasPrefix(nm, "fe"):
				fallthrough
			case strings.HasPrefix(nm, "path-effect"):
				fallthrough
			case strings.HasPrefix(nm, "filter"):
				curPar = NewFilter(curPar)
				md := curPar.(*Filter)
				md.Class = nm
				md.FilterType = nm
				for _, attr := range se.Attr {
					if SetStandardXMLAttr(md, attr.Name.Local, attr.Value) {
						continue
					}
					switch attr.Name.Local {
					default:
						curPar.AsTree().SetProperty(attr.Name.Local, attr.Value)
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
				if curPar == sv.Root.This {
					break
				}
				if curPar.AsTree().Parent == nil {
					break
				}
				curPar = curPar.AsTree().Parent.(Node)
				if curPar == sv.Root.This {
					break
				}
				r := tree.ParentByType[*Root](curPar)
				if r != nil {
					curSvg = r
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
				cp := curCSS.CSSProperties()
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
	if err != nil {
		log.Println(err)
		return err
	}
	defer fp.Close()
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

// InkscapeProperties are property keys that should be prefixed with "inkscape:"
var InkscapeProperties = map[string]bool{
	"isstock": true,
	"stockid": true,
}

// MarshalXML encodes just the given node under SVG to XML.
// It returns the name of node, for end tag; if empty, then children will not be
// output.
func MarshalXML(n tree.Node, enc *XMLEncoder, setName string) string {
	if n == nil || n.AsTree().This == nil {
		return ""
	}
	se := xml.StartElement{}
	properties := n.AsTree().Properties
	if n.AsTree().Name != "" {
		XMLAddAttr(&se.Attr, "id", n.AsTree().Name)
	}
	text := "" // if non-empty, contains text to render
	_, issvg := n.(Node)
	_, isgp := n.(*Group)
	_, ismark := n.(*Marker)
	if !isgp {
		if issvg && !ismark {
			sp := styles.StylePropertiesXML(properties)
			if sp != "" {
				XMLAddAttr(&se.Attr, "style", sp)
			}
			if txp, has := properties["transform"]; has {
				XMLAddAttr(&se.Attr, "transform", reflectx.ToString(txp))
			}
		} else {
			for k, v := range properties {
				sv := reflectx.ToString(v)
				if _, has := InkscapeProperties[k]; has {
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
	switch nd := n.(type) {
	case *Path:
		nm = "path"
		nd.DataStr = PathDataString(nd.Data)
		XMLAddAttr(&se.Attr, "d", nd.DataStr)
	case *Group:
		nm = "g"
		if strings.HasPrefix(strings.ToLower(n.AsTree().Name), "layer") {
		}
		for k, v := range properties {
			sv := reflectx.ToString(v)
			switch k {
			case "opacity", "transform":
				XMLAddAttr(&se.Attr, k, sv)
			case "groupmode":
				XMLAddAttr(&se.Attr, "inkscape:groupmode", sv)
				if st, has := properties["style"]; has {
					XMLAddAttr(&se.Attr, "style", reflectx.ToString(st))
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
		XMLAddAttr(&se.Attr, "r", fmt.Sprintf("%g", nd.Radius))
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
		XMLAddAttr(&se.Attr, "preserveAspectRatio", nd.ViewBox.PreserveAspectRatio.String())
		ib, fmt := imagex.ToBase64PNG(nd.Pixels)
		XMLAddAttr(&se.Attr, "href", "data:"+fmt+";base64,"+string(imagex.Base64SplitLines(ib)))
	case *MetaData:
		if strings.HasPrefix(nd.Name, "namedview") {
			nm = "sodipodi:namedview"
		} else if strings.HasPrefix(nd.Name, "grid") {
			nm = "inkscape:grid"
		}
	case *Gradient:
		MarshalXMLGradient(nd, nd.Name, enc)
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
		nm = n.AsTree().NodeType().Name
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

// MarshalXMLGradient adds the XML for the given gradient to the given encoder.
// This is not in [cogentcore.org/core/colors/gradient] because it uses a lot of SVG
// and XML infrastructure defined here.
func MarshalXMLGradient(n *Gradient, name string, enc *XMLEncoder) {
	gr := n.Grad
	if gr == nil {
		return
	}
	gb := gr.AsBase()
	me := xml.StartElement{}
	XMLAddAttr(&me.Attr, "id", name)

	linear := true
	if _, ok := gr.(*gradient.Radial); ok {
		linear = false
		me.Name.Local = "radialGradient"
	} else {
		me.Name.Local = "linearGradient"
	}

	if linear {
		// must be non-zero to add
		if gb.Box != (math32.Box2{}) {
			XMLAddAttr(&me.Attr, "x1", fmt.Sprintf("%g", gb.Box.Min.X))
			XMLAddAttr(&me.Attr, "y1", fmt.Sprintf("%g", gb.Box.Min.Y))
			XMLAddAttr(&me.Attr, "x2", fmt.Sprintf("%g", gb.Box.Max.X))
			XMLAddAttr(&me.Attr, "y2", fmt.Sprintf("%g", gb.Box.Max.Y))
		}
	} else {
		r := gr.(*gradient.Radial)
		// must be non-zero to add
		if r.Center != (math32.Vector2{}) {
			XMLAddAttr(&me.Attr, "cx", fmt.Sprintf("%g", r.Center.X))
			XMLAddAttr(&me.Attr, "cy", fmt.Sprintf("%g", r.Center.Y))
		}
		if r.Focal != (math32.Vector2{}) {
			XMLAddAttr(&me.Attr, "fx", fmt.Sprintf("%g", r.Focal.X))
			XMLAddAttr(&me.Attr, "fy", fmt.Sprintf("%g", r.Focal.Y))
		}
		if r.Radius != (math32.Vector2{}) {
			XMLAddAttr(&me.Attr, "r", fmt.Sprintf("%g", max(r.Radius.X, r.Radius.Y)))
		}
	}
	XMLAddAttr(&me.Attr, "gradientUnits", gb.Units.String())
	// pad is default
	if gb.Spread != gradient.Pad {
		XMLAddAttr(&me.Attr, "spreadMethod", gb.Spread.String())
	}

	if gb.Transform != math32.Identity2() {
		XMLAddAttr(&me.Attr, "gradientTransform", fmt.Sprintf("matrix(%g,%g,%g,%g,%g,%g)", gb.Transform.XX, gb.Transform.YX, gb.Transform.XY, gb.Transform.YY, gb.Transform.X0, gb.Transform.Y0))
	}

	if n.StopsName != "" {
		XMLAddAttr(&me.Attr, "href", "#"+n.StopsName)
	}

	enc.EncodeToken(me)
	if n.StopsName == "" {
		for _, gs := range gb.Stops {
			se := xml.StartElement{}
			se.Name.Local = "stop"
			clr := gs.Color
			hs := colors.AsHex(clr)[:7] // get rid of transparency
			XMLAddAttr(&se.Attr, "style", fmt.Sprintf("stop-color:%s;stop-opacity:%g;", hs, float32(colors.AsRGBA(clr).A)/255))
			XMLAddAttr(&se.Attr, "offset", fmt.Sprintf("%g", gs.Pos))
			enc.EncodeToken(se)
			enc.WriteEnd(se.Name.Local)
		}
	}
	enc.WriteEnd(me.Name.Local)
}

// MarshalXMLTree encodes the given node and any children to XML.
// It returns any error, and name of element that enc.WriteEnd() should be
// called with; allows for extra elements to be added at end of list.
func MarshalXMLTree(n Node, enc *XMLEncoder, setName string) (string, error) {
	name := MarshalXML(n, enc, setName)
	if name == "" {
		return "", nil
	}
	for _, k := range n.AsTree().Children {
		knm, err := MarshalXMLTree(k.(Node), enc, "")
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
	// TODO: what makes sense for PhysicalWidth and PhysicalHeight here?
	if sv.PhysicalWidth.Value > 0 {
		XMLAddAttr(&me.Attr, "width", fmt.Sprintf("%g", sv.PhysicalWidth.Value))
	}
	if sv.PhysicalHeight.Value > 0 {
		XMLAddAttr(&me.Attr, "height", fmt.Sprintf("%g", sv.PhysicalHeight.Value))
	}
	XMLAddAttr(&me.Attr, "viewBox", fmt.Sprintf("%g %g %g %g", sv.Root.ViewBox.Min.X, sv.Root.ViewBox.Min.Y, sv.Root.ViewBox.Size.X, sv.Root.ViewBox.Size.Y))
	XMLAddAttr(&me.Attr, "xmlns:inkscape", "http://www.inkscape.org/namespaces/inkscape")
	XMLAddAttr(&me.Attr, "xmlns:sodipodi", "http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd")
	XMLAddAttr(&me.Attr, "xmlns", "http://www.w3.org/2000/svg")
	enc.EncodeToken(me)

	dnm, err := MarshalXMLTree(sv.Defs, enc, "defs")
	enc.WriteEnd(dnm)

	for _, k := range sv.Root.Children {
		var knm string
		knm, err = MarshalXMLTree(k.(Node), enc, "")
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

// SetStandardXMLAttr sets standard attributes of node given XML-style name /
// attribute values (e.g., from parsing XML / SVG files); returns true if handled.
func SetStandardXMLAttr(ni Node, name, val string) bool {
	nb := ni.AsNodeBase()
	switch name {
	case "id":
		nb.SetName(val)
		return true
	case "class":
		nb.Class = val
		return true
	case "style":
		styles.SetStylePropertiesXML(val, (*map[string]any)(&nb.Properties))
		return true
	}
	return false
}
