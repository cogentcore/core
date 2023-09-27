// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// color parsing is adapted from github.com/srwiley/oksvg:
//
// Copyright 2017 The oksvg Authors. All rights reserved.
//
// created: 2/12/2017 by S.R.Wiley

package colors

import (
	"encoding/xml"
	"fmt"
	"image/color"
	"io"
	"log"
	"strconv"
	"strings"

	"goki.dev/colors"
	"goki.dev/laser"
	"goki.dev/mat32/v2"

	"github.com/srwiley/rasterx"
	"golang.org/x/net/html/charset"
)

// XMLAttr searches for given attribute in slice of xml attributes -- returns "" if not found
func XMLAttr(name string, attrs []xml.Attr) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

// FullCache is a cache of named full colors -- only a few are constantly re-used
// so we save them in the cache instead of constantly recomputing!
var FullCache map[string]*Full

// SetString sets the color spec from a standard CSS-formatted string.
// SetString is based on https://www.w3schools.com/css/css3_gradients.asp.
// See [Full.UnmarshalXML] for an XML-based version.
func (f *Full) SetString(str string, base color.Color) bool {
	if FullCache == nil {
		FullCache = make(map[string]*Full)
	}
	fullnm := colors.AsHex(f.Solid) + str
	if ccg, ok := FullCache[fullnm]; ok {
		f.CopyFrom(ccg)
		return true
	}

	str = strings.TrimSpace(str)
	if strings.HasPrefix(str, "url(") {
		if ctxt != nil {
			cspec := ctxt.ContextColorSpecByURL(str)
			if cspec != nil {
				*f = *cspec
				return true
			}
		}
		fmt.Printf("gi.Color Warning: Not able to find url: %v\n", str)
		f.Gradient = nil
		f.Solid = Black
		return false
	}
	str = strings.ToLower(str)
	grad := "-gradient"
	if gidx := strings.Index(str, grad); gidx > 0 {
		gtyp := str[:gidx]
		rmdr := str[gidx+len(grad):]
		pidx := strings.IndexByte(rmdr, '(')
		if pidx < 0 {
			log.Printf("gi.ColorSpec.Parse gradient parameters not found\n")
			return false
		}
		pars := rmdr[pidx+1:]
		pars = strings.TrimSuffix(pars, ");")
		pars = strings.TrimSuffix(pars, ")")
		switch gtyp {
		case "repeating-linear":
			f.Gradient = LinearGradient().SetSpread(RepeatSpread)
			f.parseLinearGrad(pars)
		case "linear":
			f.Gradient = LinearGradient()
			f.parseLinearGrad(pars)
		case "repeating-radial":
			f.Gradient = RadialGradient().SetSpread(RepeatSpread)
			f.parseRadialGrad(pars)
		case "radial":
			f.Gradient = RadialGradient()
			f.parseRadialGrad(pars)
		}
		FixGradientStops(f.Gradient)
		svcs := &Full{} // critical to save a copy..
		svcs.CopyFrom(f)
		FullCache[fullnm] = svcs
	} else {
		f.Gradient = nil
		f.Solid = colors.LogFromString(str, nil)
	}
	return true
}

// GradientDegToSides maps gradient degree notation to side notation
var GradientDegToSides = map[string]string{
	"0deg":    "top",
	"360deg":  "top",
	"45deg":   "top right",
	"-315deg": "top right",
	"90deg":   "right",
	"-270deg": "right",
	"135deg":  "bottom right",
	"-225deg": "bottom right",
	"180deg":  "bottom",
	"-180deg": "bottom",
	"225deg":  "bottom left",
	"-135deg": "bottom left",
	"270deg":  "left",
	"-90deg":  "left",
	"315deg":  "top left",
	"-45deg":  "top left",
}

func (f *Full) parseLinearGrad(pars string) bool {
	plist := strings.Split(pars, ", ")
	var prevColor color.RGBA
	stopIdx := 0
	for pidx := 0; pidx < len(plist); pidx++ {
		par := strings.TrimRight(strings.TrimSpace(plist[pidx]), ",")
		origPar := par
		switch {
		case strings.Contains(par, "deg"):
			// can't use trig, b/c need to be full 1, 0 values -- just use map
			var ok bool
			par, ok = GradientDegToSides[par]
			if !ok {
				log.Printf("gi.ColorSpec.Parse invalid gradient angle -- must be at 45 degree increments: %v\n", origPar)
			}
			par = "to " + par
			fallthrough
		case strings.HasPrefix(par, "to "):
			sides := strings.Split(par[3:], " ")
			f.Gradient.Bounds = mat32.Box2{}
			for _, side := range sides {
				switch side {
				case "bottom":
					f.Gradient.Bounds.Min.Y = 0
					f.Gradient.Bounds.Max.Y = 1
				case "top":
					f.Gradient.Bounds.Min.Y = 1
					f.Gradient.Bounds.Max.Y = 0
				case "right":
					f.Gradient.Bounds.Min.X = 0
					f.Gradient.Bounds.Max.X = 1
				case "left":
					f.Gradient.Bounds.Min.X = 1
					f.Gradient.Bounds.Max.X = 0
				}
			}
		case strings.HasPrefix(par, ")"):
			break
		default: // must be a color stop
			var stop *GradientStop
			if len(f.Gradient.Stops) > stopIdx {
				stop = &(f.Gradient.Stops[stopIdx])
			} else {
				stop = &GradientStop{Opacity: 1.0, Color: Black}
			}
			if stopIdx == 0 {
				prevColor = f.Solid // base color
				// fmt.Printf("starting prev color: %v\n", prevColor)
			}
			if parseColorStop(stop, prevColor, par) {
				if len(f.Gradient.Stops) <= stopIdx {
					f.Gradient.Stops = append(f.Gradient.Stops, *stop)
				}
				if stopIdx == 0 {
					f.Solid = colors.AsRGBA(stop.Color) // keep first one
				}
				prevColor = stop.Color
				stopIdx++
			}
		}
	}
	if len(f.Gradient.Stops) > stopIdx {
		f.Gradient.Stops = f.Gradient.Stops[:stopIdx]
	}
	return true
}

// todo: this is complex:
// https://www.w3.org/TR/css3-images/#radial-gradients

func (f *Full) parseRadialGrad(pars string) bool {
	plist := strings.Split(pars, ", ")
	var prevColor color.RGBA
	stopIdx := 0
	for pidx := 0; pidx < len(plist); pidx++ {
		par := strings.TrimRight(strings.TrimSpace(plist[pidx]), ",")
		// origPar := par
		switch {
		case strings.Contains(par, "circle"):
			f.Gradient.Center.SetScalar(0.5)
			f.Gradient.Focal.SetScalar(0.5)
			f.Gradient.Radius = 0.5
		case strings.Contains(par, "ellipse"):
			f.Gradient.Center.SetScalar(0.5)
			f.Gradient.Focal.SetScalar(0.5)
			f.Gradient.Radius = 0.5
		case strings.HasPrefix(par, "at "):
			sides := strings.Split(par[3:], " ")
			f.Gradient.Center = mat32.Vec2{}
			f.Gradient.Focal = mat32.Vec2{}
			f.Gradient.Radius = 0
			for _, side := range sides {
				switch side {
				case "bottom":
					f.Gradient.Bounds.Min.Y = 0
				case "top":
					f.Gradient.Bounds.Min.Y = 1
				case "right":
					f.Gradient.Bounds.Min.X = 0
				case "left":
					f.Gradient.Bounds.Min.X = 1
				}
			}
		case strings.HasPrefix(par, ")"):
			break
		default: // must be a color stop
			var stop *GradientStop
			if len(f.Gradient.Stops) > stopIdx {
				stop = &(f.Gradient.Stops[stopIdx])
			} else {
				stop = &GradientStop{Opacity: 1.0, Color: Black}
			}
			if stopIdx == 0 {
				prevColor = f.Solid // base color
				// fmt.Printf("starting prev color: %v\n", prevColor)
			}
			if parseColorStop(stop, prevColor, par) {
				if len(f.Gradient.Stops) <= stopIdx {
					f.Gradient.Stops = append(f.Gradient.Stops, *stop)
				}
				if stopIdx == 0 {
					f.Solid = colors.AsRGBA(stop.Color) // keep first one
				}
				prevColor = stop.Color
				stopIdx++
			}
		}
	}
	if len(f.Gradient.Stops) > stopIdx {
		f.Gradient.Stops = f.Gradient.Stops[:stopIdx]
	}
	return true
}

func parseColorStop(stop *GradientStop, prevColor color.RGBA, par string) bool {
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
	// color blending doesn't work well in pre-multiplied alpha RGB space!
	if IsNil(prevColor) && strings.HasPrefix(cnm, "clearer-") {
		pcts := strings.TrimPrefix(cnm, "clearer-")
		pct, _ := laser.ToFloat(pcts)
		stop.Opacity = (100.0 - float32(pct)) / 100.0
		stop.Color = prevColor
	} else if IsNil(prevColor) && cnm == "transparent" {
		stop.Opacity = 0
		stop.Color = prevColor
	} else {
		clr, err := colors.FromString(cnm, prevColor)
		if err != nil {
			log.Printf("gi.ColorSpec.Parse invalid color string: %v\n", err)
			return false
		}
		stop.Color = clr
	}
	// fmt.Printf("color: %v from: %v\n", stop.StopColor, par)
	return true
}

// ReadXML reads XML-formatted ColorSpec from io.Reader
func (cs *ColorSpec) ReadXML(reader io.Reader) error {
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel
	for {
		t, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("gi.ColorSpec parsing error: %v\n", err)
			return err
		}
		switch se := t.(type) {
		case xml.StartElement:
			return cs.UnmarshalXML(decoder, se)
			// todo: ignore rest?
		}
	}
	return nil
}

// UnmarshalXML parses the given XML-formatted string to set the color
// specification
func (cs *ColorSpec) UnmarshalXML(decoder *xml.Decoder, se xml.StartElement) error {
	start := &se

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
			log.Printf("gi.ColorSpec.UnmarshalXML color parsing error: %v\n", err)
			return err
		}
		switch se := t.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "linearGradient":
				if cs.Gradient == nil {
					cs.Gradient = &rasterx.Gradient{Points: [5]float64{0, 0, 1, 0, 0},
						IsRadial: false, Matrix: rasterx.Identity}
				} else {
					cs.Gradient.IsRadial = false
				}
				cs.Source = LinearGradient
				// fmt.Printf("lingrad %v\n", cs.Gradient)
				for _, attr := range se.Attr {
					// fmt.Printf("attr: %v val: %v\n", attr.Name.Local, attr.Value)
					switch attr.Name.Local {
					// note: id not processed here - must be done externally
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
						log.Printf("gi.ColorSpec.UnmarshalXML linear gradient parsing error: %v\n", err)
						return err
					}
				}
			case "radialGradient":
				if cs.Gradient == nil {
					cs.Gradient = &rasterx.Gradient{Points: [5]float64{0.5, 0.5, 0.5, 0.5, 0.5},
						IsRadial: true, Matrix: rasterx.Identity}
				} else {
					cs.Gradient.IsRadial = true
				}
				cs.Source = RadialGradient
				var setFx, setFy bool
				for _, attr := range se.Attr {
					// fmt.Printf("stop attr: %v val: %v\n", attr.Name.Local, attr.Value)
					switch attr.Name.Local {
					// note: id not processed here - must be done externally
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
						log.Printf("gi.ColorSpec.UnmarshalXML radial gradient parsing error: %v\n", err)
						return err
					}
				}
				if setFx == false { // set fx to cx by default
					cs.Gradient.Points[2] = cs.Gradient.Points[0]
				}
				if setFy == false { // set fy to cy by default
					cs.Gradient.Points[3] = cs.Gradient.Points[1]
				}
			case "stop":
				stop := rasterx.GradStop{Opacity: 1.0, StopColor: colors.Black}
				ats := se.Attr
				sty := XMLAttr("style", ats)
				if sty != "" {
					spl := strings.Split(sty, ";")
					for _, s := range spl {
						s := strings.TrimSpace(s)
						ci := strings.IndexByte(s, ':')
						if ci < 0 {
							continue
						}
						a := xml.Attr{}
						a.Name.Local = s[:ci]
						a.Value = s[ci+1:]
						ats = append(ats, a)
					}
				}
				for _, attr := range ats {
					switch attr.Name.Local {
					case "offset":
						stop.Offset, err = readFraction(attr.Value)
					case "stop-color":
						clr, err := colors.FromString(attr.Value, nil)
						if err != nil {
							log.Printf("gi.ColorSpec.UnmarshalXML invalid color string: %v\n", err)
							return err
						}
						stop.StopColor = clr
					case "stop-opacity":
						stop.Opacity, err = strconv.ParseFloat(attr.Value, 64)
						// fmt.Printf("got opacity: %v\n", stop.Opacity)
					}
					if err != nil {
						log.Printf("gi.ColorSpec.UnmarshalXML color stop parsing error: %v\n", err)
						return err
					}
				}
				if cs.Gradient == nil {
					fmt.Printf("no gradient but stops in: %v stop: %v\n", cs, stop)
				} else {
					cs.Gradient.Stops = append(cs.Gradient.Stops, stop)
				}
			default:
				errStr := "gi.ColorSpec Cannot process svg element " + se.Name.Local
				log.Println(errStr)
			}
		case xml.EndElement:
			if se.Name.Local == "linearGradient" || se.Name.Local == "radialGradient" {
				// fmt.Printf("gi.ColorSpec got gradient end element: %v\n", se.Name.Local)
				return nil
			}
			if se.Name.Local != "stop" {
				fmt.Printf("gi.ColorSpec got unexpected end element: %v\n", se.Name.Local)
			}
		case xml.CharData:
		}
	}
	return nil
}

func readFraction(v string) (f float32, err error) {
	v = strings.TrimSpace(v)
	d := 1.0
	if strings.HasSuffix(v, "%") {
		d = 100
		v = strings.TrimSuffix(v, "%")
	}
	f64, err := strconv.ParseFloat(v, 64)
	f64 /= d
	// if f > 1 {
	// 	f = 1
	// } else
	if f < 0 {
		f = 0
	}
	return float32(f64), err
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
		tx := mat32.Identity2D()
		tx.SetString(attr.Value)
		cs.Gradient.Matrix = MatToRasterx(&tx)
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

// FixGradientStops applies the CSS rules to regularize the gradient stops: https://www.w3.org/TR/css3-images/#color-stop-syntax
func FixGradientStops(grad *rasterx.Gradient) {
	sz := len(grad.Stops)
	if sz == 0 {
		return
	}
	splitSt := -1
	last := 0.0
	for i := 0; i < sz; i++ {
		st := &(grad.Stops[i])
		if i == sz-1 && st.Offset == 0 {
			if last < 1.0 {
				st.Offset = 1.0
			} else {
				st.Offset = last
			}
		}
		if i > 0 && st.Offset == 0 && splitSt < 0 {
			splitSt = i
			st.Offset = last
			continue
		}
		if splitSt > 0 {
			start := grad.Stops[splitSt].Offset
			end := st.Offset
			per := (end - start) / float64(1+(i-splitSt))
			cur := start + per
			for j := splitSt; j < i; j++ {
				grad.Stops[j].Offset = cur
				cur += per
			}
		}
		last = st.Offset
	}
	// fmt.Printf("grad stops:\n")
	// for i := 0; i < sz; i++ {
	// 	st := grad.Stops[i]
	// 	fmt.Printf("%v\t%v opacity: %v\n", i, st.Offset, st.Opacity)
	// }
}
