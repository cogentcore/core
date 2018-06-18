// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  SVG

// SVG is a viewport for containing SVG drawing objects, correponding to the
// svg tag in html -- it provides its own bitmap for drawing into
type SVG struct {
	Viewport2D
	Defs  Layout `desc:"all defs defined elements go here (gradients, symbols, etc)"`
	Title string `xml:"title" desc:"the title of the svg"`
	Desc  string `xml:"desc" desc:"the description of the svg"`
}

var KiT_SVG = kit.Types.AddType(&SVG{}, nil)

// set a normalized 0-1 scaling transform so svg's use 0-1 coordinates that
// map to actual size of the viewport -- used e.g. for Icon
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

////////////////////////////////////////////////////////////////////////////////////////
//  todo parsing code etc

// ParseXML parses the given XML-formatted string to set the color
// specification -- recognizes svg and css gradient specifications -- tree is
// used to find url references if non-nil
// func (svg *SVG) ParseXML() bool {
// 	decoder := xml.NewDecoder(bytes.NewReader([]byte(clrstr)))
// 	decoder.CharsetReader = charset.NewReaderLabel
// 	for {
// 		t, err := decoder.Token()
// 		if err != nil {
// 			if err == io.EOF {
// 				break
// 			}
// 			log.Printf("gi.ColorSpec.ParseXML color parsing error: %v\n", err)
// 			return false
// 		}
// 		switch se := t.(type) {
// 		case xml.StartElement:
// 			switch se.Name.Local {
// 			case "linearGradient":
// 				cs.Gradient = &rasterx.Gradient{Points: [5]float64{0, 0, 0, 1, 0},
// 					IsRadial: false, Matrix: rasterx.Identity}
// 				for _, attr := range se.Attr {
// 					switch attr.Name.Local {
// 					case "id":
// 						// id := attr.Value
// 						// if len(id) >= 0 {
// 						// 	icon.Ids[id] = cursor.grad
// 						// } else {
// 						// 	return icon, zeroLengthIdError
// 						// }
// 					case "x1":
// 						cs.Gradient.Points[0], err = readFraction(attr.Value)
// 					case "y1":
// 						cs.Gradient.Points[1], err = readFraction(attr.Value)
// 					case "x2":
// 						cs.Gradient.Points[2], err = readFraction(attr.Value)
// 					case "y2":
// 						cs.Gradient.Points[3], err = readFraction(attr.Value)
// 					default:
// 						err = cs.ReadGradAttr(attr)
// 					}
// 					if err != nil {
// 						log.Printf("gi.ColorSpec.ParseXML linear gradient parsing error: %v\n", err)
// 						return false
// 					}
// 				}
// 			case "radialGradient":
// 				cs.Gradient = &rasterx.Gradient{Points: [5]float64{0.5, 0.5, 0.5, 0.5, 0.5},
// 					IsRadial: true, Matrix: rasterx.Identity}
// 				var setFx, setFy bool
// 				for _, attr := range se.Attr {
// 					switch attr.Name.Local {
// 					case "id":
// 						// id := attr.Value
// 						// if len(id) >= 0 {
// 						// 	icon.Ids[id] = cursor.grad
// 						// } else {
// 						// 	return icon, zeroLengthIdError
// 						// }
// 					case "r":
// 						cs.Gradient.Points[4], err = readFraction(attr.Value)
// 					case "cx":
// 						cs.Gradient.Points[0], err = readFraction(attr.Value)
// 					case "cy":
// 						cs.Gradient.Points[1], err = readFraction(attr.Value)
// 					case "fx":
// 						setFx = true
// 						cs.Gradient.Points[2], err = readFraction(attr.Value)
// 					case "fy":
// 						setFy = true
// 						cs.Gradient.Points[3], err = readFraction(attr.Value)
// 					default:
// 						err = cs.ReadGradAttr(attr)
// 					}
// 					if err != nil {
// 						log.Printf("gi.ColorSpec.ParseXML radial gradient parsing error: %v\n", err)
// 						return false
// 					}
// 				}
// 				if setFx == false { // set fx to cx by default
// 					cs.Gradient.Points[2] = cs.Gradient.Points[0]
// 				}
// 				if setFy == false { // set fy to cy by default
// 					cs.Gradient.Points[3] = cs.Gradient.Points[1]
// 				}
// 			case "stop":
// 				stop := rasterx.GradStop{Opacity: 1.0}
// 				for _, attr := range se.Attr {
// 					switch attr.Name.Local {
// 					case "offset":
// 						stop.Offset, err = readFraction(attr.Value)
// 					case "stop-color":
// 						clr, err := ColorFromString(attr.Value, nil)
// 						if err != nil {
// 							log.Printf("gi.ColorSpec.ParseXML invalid color string: %v\n", err)
// 							return false
// 						}
// 						stop.StopColor = clr
// 					case "stop-opacity":
// 						stop.Opacity, err = strconv.ParseFloat(attr.Value, 64)
// 					}
// 					if err != nil {
// 						log.Printf("gi.ColorSpec.ParseXML color stop parsing error: %v\n", err)
// 						return false
// 					}
// 				}
// 				cs.Gradient.Stops = append(cs.Gradient.Stops, &stop)
// 			default:
// 				errStr := "Cannot process svg element " + se.Name.Local
// 				log.Println(errStr)
// 			}
// 		case xml.EndElement:
// 		case xml.CharData:
// 		}
// 	}
// 	return true
// }
