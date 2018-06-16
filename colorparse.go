// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// color parsing is adapted from github.com/srwiley/oksvg:
//
// Copyright 2017 The oksvg Authors. All rights reserved.
//
// created: 2/12/2017 by S.R.Wiley

package gi

import (
	"bytes"
	"encoding/xml"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/goki/ki"
	"github.com/srwiley/rasterx"
	"golang.org/x/net/html/charset"
)

// Parse parses the given color string to set the color specification --
// recognizes svg (XML) and css gradient specifications -- tree is used to find url
// references if non-nil
func (cs *ColorSpec) Parse(clrstr string, tree ki.Ki) bool {
	clrstr = strings.TrimSpace(clrstr)
	if strings.HasPrefix(clrstr, "<") {
		return cs.ParseXML(clrstr, tree)
	}
	clrstr = strings.ToLower(clrstr)
	grad := "-gradient"
	if gidx := strings.Index(clrstr, grad); gidx > 0 {
		gtyp := clrstr[:gidx]
		rmdr := clrstr[gidx+len(grad):]
		pidx := strings.IndexRune(rmdr, '(')
		if pidx < 0 {
			log.Printf("gi.ColorSpec.Parse gradient parameters not found\n")
			return false
		}
		cs.Gradient = &rasterx.Gradient{Points: [5]float64{0, 0, 0, 1, 0},
			IsRadial: false, Matrix: rasterx.Identity}
		pars := rmdr[pidx+1:]
		switch gtyp {
		case "repeating-linear":
			cs.Source = LinearGradient
			cs.Gradient.IsRadial = false
			cs.Gradient.Spread = rasterx.RepeatSpread
			cs.parseLinearGrad(pars)
		case "linear":
			cs.Source = LinearGradient
			cs.Gradient.IsRadial = false
			cs.Gradient.Spread = rasterx.PadSpread
			cs.parseLinearGrad(pars)
		}
		FixGradientStops(cs.Gradient)
	} else {
		cs.Gradient = nil
		cs.Source = SolidColor
		cs.Color.SetString(clrstr, nil)
	}
	return true
}

// GradientDegToSides maps gradient degree notation to side notation
var GradientDegToSides = map[string]string{
	"0deg":    "up",
	"360deg":  "up",
	"45deg":   "up right",
	"-315deg": "up right",
	"90deg":   "right",
	"-270deg": "right",
	"135deg":  "down right",
	"-225deg": "down right",
	"180deg":  "down",
	"-180deg": "down",
	"225deg":  "down left",
	"-135deg": "down left",
	"270deg":  "left",
	"-90deg":  "left",
	"315deg":  "up left",
	"-45deg":  "up left",
}

func (cs *ColorSpec) parseLinearGrad(pars string) bool {
	plist := strings.Split(pars, ",")
	pidx := 0
	for pidx < len(plist) {
		par := strings.TrimSpace(plist[pidx])
		origPar := par
		switch {
		case strings.Contains(par, "deg"):
			// can't use trig, b/c need to be full 1, 0 values -- just use map
			var ok bool
			par, ok = GradientDegToSides[par]
			if !ok {
				log.Printf("gi.ColorSpec.Parse invalid gradient angle -- must be at 45 degree increments: %v\n", origPar)
			}
			fallthrough
		case strings.HasPrefix(par, "to "):
			sides := strings.Split(strings.TrimLeft(par, "to "), " ")
			cs.Gradient.Points = [5]float64{0, 0, 0, 0, 0}
			for _, side := range sides {
				switch side {
				case "bottom":
					cs.Gradient.Points[GpY1] = 0
					cs.Gradient.Points[GpY2] = 1
				case "top":
					cs.Gradient.Points[GpY1] = 1
					cs.Gradient.Points[GpY2] = 0
				case "right":
					cs.Gradient.Points[GpX1] = 0
					cs.Gradient.Points[GpX2] = 1
				case "left":
					cs.Gradient.Points[GpX1] = 1
					cs.Gradient.Points[GpX2] = 0
				}
			}
		default: // must be a color stop
			stop := rasterx.GradStop{Opacity: 1.0}
			if parseColorStop(&stop, par) {
				cs.Gradient.Stops = append(cs.Gradient.Stops, stop)
			}
		}
	}
	return true
}

func parseColorStop(stop *rasterx.GradStop, par string) bool {
	cnm := par
	if spcidx := strings.Index(par, " "); spcidx > 0 {
		cnm = par[:spcidx]
		offs := strings.TrimSpace(par[spcidx+1:])
		off, err := readFraction(offs)
		if err != nil {
			log.Printf("gi.ColorSpec.Parse invalid offset: %v\n", err)
			return false
		}
		stop.Offset = off
	}
	clr, err := ColorFromString(cnm, nil)
	if err != nil {
		log.Printf("gi.ColorSpec.Parse invalid color string: %v\n", err)
		return false
	}
	stop.StopColor = clr
	return true
}

// ParseXL parses the given XML-formatted string to set the color
// specification -- recognizes svg and css gradient specifications -- tree is
// used to find url references if non-nil
func (cs *ColorSpec) ParseXML(clrstr string, tree ki.Ki) bool {
	decoder := xml.NewDecoder(bytes.NewReader([]byte(clrstr)))
	decoder.CharsetReader = charset.NewReaderLabel
	for {
		t, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("gi.ColorSpec.ParseXML color parsing error: %v\n", err)
			return false
		}
		switch se := t.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "linearGradient":
				cs.Gradient = &rasterx.Gradient{Points: [5]float64{0, 0, 0, 1, 0},
					IsRadial: false, Matrix: rasterx.Identity}
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					case "id":
						// id := attr.Value
						// if len(id) >= 0 {
						// 	icon.Ids[id] = cursor.grad
						// } else {
						// 	return icon, zeroLengthIdError
						// }
					case "x1":
						cs.Gradient.Points[0], err = readFraction(attr.Value)
					case "y1":
						cs.Gradient.Points[1], err = readFraction(attr.Value)
					case "x2":
						cs.Gradient.Points[2], err = readFraction(attr.Value)
					case "y2":
						cs.Gradient.Points[3], err = readFraction(attr.Value)
					default:
						err = cs.ReadGradAttr(attr)
					}
					if err != nil {
						log.Printf("gi.ColorSpec.ParseXML linear gradient parsing error: %v\n", err)
						return false
					}
				}
			case "radialGradient":
				cs.Gradient = &rasterx.Gradient{Points: [5]float64{0.5, 0.5, 0.5, 0.5, 0.5},
					IsRadial: true, Matrix: rasterx.Identity}
				var setFx, setFy bool
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					case "id":
						// id := attr.Value
						// if len(id) >= 0 {
						// 	icon.Ids[id] = cursor.grad
						// } else {
						// 	return icon, zeroLengthIdError
						// }
					case "r":
						cs.Gradient.Points[4], err = readFraction(attr.Value)
					case "cx":
						cs.Gradient.Points[0], err = readFraction(attr.Value)
					case "cy":
						cs.Gradient.Points[1], err = readFraction(attr.Value)
					case "fx":
						setFx = true
						cs.Gradient.Points[2], err = readFraction(attr.Value)
					case "fy":
						setFy = true
						cs.Gradient.Points[3], err = readFraction(attr.Value)
					default:
						err = cs.ReadGradAttr(attr)
					}
					if err != nil {
						log.Printf("gi.ColorSpec.ParseXML radial gradient parsing error: %v\n", err)
						return false
					}
				}
				if setFx == false { // set fx to cx by default
					cs.Gradient.Points[2] = cs.Gradient.Points[0]
				}
				if setFy == false { // set fy to cy by default
					cs.Gradient.Points[3] = cs.Gradient.Points[1]
				}
			case "stop":
				stop := rasterx.GradStop{Opacity: 1.0}
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					case "offset":
						stop.Offset, err = readFraction(attr.Value)
					case "stop-color":
						clr, err := ColorFromString(attr.Value, nil)
						if err != nil {
							log.Printf("gi.ColorSpec.ParseXML invalid color string: %v\n", err)
							return false
						}
						stop.StopColor = clr
					case "stop-opacity":
						stop.Opacity, err = strconv.ParseFloat(attr.Value, 64)
					}
					if err != nil {
						log.Printf("gi.ColorSpec.ParseXML color stop parsing error: %v\n", err)
						return false
					}
				}
				cs.Gradient.Stops = append(cs.Gradient.Stops, stop)
			default:
				errStr := "Cannot process svg element " + se.Name.Local
				log.Println(errStr)
			}
		case xml.EndElement:
		case xml.CharData:
		}
	}
	return true
}

func readFraction(v string) (f float64, err error) {
	v = strings.TrimSpace(v)
	d := 1.0
	if strings.HasSuffix(v, "%") {
		d = 100
		v = strings.TrimSuffix(v, "%")
	}
	f, err = strconv.ParseFloat(v, 64)
	f /= d
	if f > 1 {
		f = 1
	} else if f < 0 {
		f = 0
	}
	return
}

// func ReadGradUrl(v string) (grad *rasterx.Gradient, err error) {
// 	if strings.HasPrefix(v, "url(") && strings.HasSuffix(v, ")") {
// 		urlStr := strings.TrimSpace(v[4 : len(v)-1])
// 		if strings.HasPrefix(urlStr, "#") {
// 			switch grad := c.icon.Ids[urlStr[1:]].(type) {
// 			case *rasterx.Gradient:
// 				return grad, nil
// 			default:
// 				return nil, nil //missingIdError
// 			}

// 		}
// 	}
// 	return nil, nil // not a gradient url, and not an error
// }

func (cs *ColorSpec) ReadGradAttr(attr xml.Attr) (err error) {
	switch attr.Name.Local {
	case "gradientTransform":
		// cs.Gradient.Matrix, err = cursor.parseTransform(attr.Value)
	case "gradientUnits":
		switch strings.TrimSpace(attr.Value) {
		case "userSpaceOnUse":
			cs.Gradient.Units = rasterx.UserSpaceOnUse
		case "objectBoundingBox":
			cs.Gradient.Units = rasterx.ObjectBoundingBox
		}
	case "spreadMethod":
		switch strings.TrimSpace(attr.Value) {
		case "pad":
			cs.Gradient.Spread = rasterx.PadSpread
		case "reflect":
			cs.Gradient.Spread = rasterx.ReflectSpread
		case "repeat":
			cs.Gradient.Spread = rasterx.RepeatSpread
		}
	}
	return nil
}

func FixGradientStops(grad *rasterx.Gradient) {
	sz := len(grad.Stops)
	splitSt := -1
	last := 0.0
	for i := 0; i < sz; i++ {
		st := grad.Stops[i]
		if i > 0 && st.Offset == 0 && splitSt < 0 {
			splitSt = i
			st.Offset = last
			continue
		} else if i == sz-1 && st.Offset == 0 {
			st.Offset = 1
		}
		if splitSt > 0 {
			start := grad.Stops[splitSt].Offset
			end := st.Offset
			per := (end - start) / float64(1+(i-splitSt))
			cur := start + per
			for j := splitSt + 1; j < i; j++ {
				grad.Stops[j].Offset = cur
				cur += per
			}
		}
		last = st.Offset
	}
}
