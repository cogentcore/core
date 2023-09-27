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
	"strconv"
	"strings"

	"goki.dev/laser"
	"goki.dev/mat32/v2"

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
func (f *Full) SetString(str string, ctx Context) error {
	if FullCache == nil {
		FullCache = make(map[string]*Full)
	}
	fullnm := AsHex(f.Solid) + str
	if ccg, ok := FullCache[fullnm]; ok {
		f.CopyFrom(ccg)
		return nil
	}

	str = strings.TrimSpace(str)
	// TODO: handle url values
	if strings.HasPrefix(str, "url(") {
		if ctx != nil {
			full := ctx.FullByURL(str)
			if full != nil {
				*f = *full
				return nil
			}
		}
		f.Gradient = nil
		f.Solid = Black
		return fmt.Errorf("unable to find url %q", str)
	}
	str = strings.ToLower(str)
	grad := "-gradient"
	if gidx := strings.Index(str, grad); gidx > 0 {
		gtyp := str[:gidx]
		rmdr := str[gidx+len(grad):]
		pidx := strings.IndexByte(rmdr, '(')
		if pidx < 0 {
			return fmt.Errorf("gradient specified but parameters not found in string %q", str)
		}
		pars := rmdr[pidx+1:]
		pars = strings.TrimSuffix(pars, ");")
		pars = strings.TrimSuffix(pars, ")")
		switch gtyp {
		case "repeating-linear":
			f.Gradient = LinearGradient().SetSpread(RepeatSpread)
			err := f.parseLinearGrad(pars)
			if err != nil {
				return err
			}
		case "linear":
			f.Gradient = LinearGradient()
			err := f.parseLinearGrad(pars)
			if err != nil {
				return err
			}
		case "repeating-radial":
			f.Gradient = RadialGradient().SetSpread(RepeatSpread)
			err := f.parseRadialGrad(pars)
			if err != nil {
				return err
			}
		case "radial":
			f.Gradient = RadialGradient()
			err := f.parseRadialGrad(pars)
			if err != nil {
				return err
			}
		}
		FixGradientStops(f.Gradient)
		svcs := &Full{} // critical to save a copy..
		svcs.CopyFrom(f)
		FullCache[fullnm] = svcs
	} else {
		f.Gradient = nil
		s, err := FromString(str, nil)
		if err != nil {
			return err
		}
		f.Solid = s
	}
	return nil
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

func (f *Full) parseLinearGrad(pars string) error {
	plist := strings.Split(pars, ", ")
	var prevColor color.RGBA
	stopIdx := 0
outer:
	for pidx := 0; pidx < len(plist); pidx++ {
		par := strings.TrimRight(strings.TrimSpace(plist[pidx]), ",")
		origPar := par
		switch {
		case strings.Contains(par, "deg"):
			// can't use trig, b/c need to be full 1, 0 values -- just use map
			var ok bool
			par, ok = GradientDegToSides[par]
			if !ok {
				return fmt.Errorf("invalid gradient angle %q: must be at 45 degree increments", origPar)
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
			break outer
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
			err := parseColorStop(stop, prevColor, par)
			if err != nil {
				return err
			}
			if len(f.Gradient.Stops) <= stopIdx {
				f.Gradient.Stops = append(f.Gradient.Stops, *stop)
			}
			if stopIdx == 0 {
				f.Solid = AsRGBA(stop.Color) // keep first one
			}
			prevColor = stop.Color
			stopIdx++
		}
	}
	if len(f.Gradient.Stops) > stopIdx {
		f.Gradient.Stops = f.Gradient.Stops[:stopIdx]
	}
	return nil
}

// todo: this is complex:
// https://www.w3.org/TR/css3-images/#radial-gradients

func (f *Full) parseRadialGrad(pars string) error {
	plist := strings.Split(pars, ", ")
	var prevColor color.RGBA
	stopIdx := 0
outer:
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
			break outer
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
			err := parseColorStop(stop, prevColor, par)
			if err != nil {
				return err
			}
			if len(f.Gradient.Stops) <= stopIdx {
				f.Gradient.Stops = append(f.Gradient.Stops, *stop)
			}
			if stopIdx == 0 {
				f.Solid = AsRGBA(stop.Color) // keep first one
			}
			prevColor = stop.Color
			stopIdx++
		}
	}
	if len(f.Gradient.Stops) > stopIdx {
		f.Gradient.Stops = f.Gradient.Stops[:stopIdx]
	}
	return nil
}

func parseColorStop(stop *GradientStop, prevColor color.RGBA, par string) error {
	cnm := par
	if spcidx := strings.Index(par, " "); spcidx > 0 {
		cnm = par[:spcidx]
		offs := strings.TrimSpace(par[spcidx+1:])
		off, err := readFraction(offs)
		if err != nil {
			return fmt.Errorf("invalid offset %q: %w", offs, err)
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
		clr, err := FromString(cnm, prevColor)
		if err != nil {
			return fmt.Errorf("invalid color string %q: %w", cnm, err)
		}
		stop.Color = clr
	}
	return nil
}

// ReadXML reads a XML-formatted [Full] from the given io.Reader
func (f *Full) ReadXML(reader io.Reader) error {
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel
	for {
		t, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error parsing color xml: %w", err)
		}
		switch se := t.(type) {
		case xml.StartElement:
			return f.UnmarshalXML(decoder, se)
			// todo: ignore rest?
		}
	}
	return nil
}

// UnmarshalXML parses the given XML-formatted string to set the color
// specification
func (f *Full) UnmarshalXML(decoder *xml.Decoder, se xml.StartElement) error {
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
			return fmt.Errorf("error parsing color: %w", err)
		}
		switch se := t.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "linearGradient":
				if f.Gradient == nil {
					f.Gradient = LinearGradient()
					f.Gradient.Bounds.Max = mat32.Vec2{1, 0} // SVG is LTR by default
				} else {
					f.Gradient.Radial = false
				}
				// fmt.Printf("lingrad %v\n", cs.Gradient)
				for _, attr := range se.Attr {
					// fmt.Printf("attr: %v val: %v\n", attr.Name.Local, attr.Value)
					switch attr.Name.Local {
					// note: id not processed here - must be done externally
					case "x1":
						f.Gradient.Bounds.Min.X, err = readFraction(attr.Value)
					case "y1":
						f.Gradient.Bounds.Min.Y, err = readFraction(attr.Value)
					case "x2":
						f.Gradient.Bounds.Max.X, err = readFraction(attr.Value)
					case "y2":
						f.Gradient.Bounds.Max.Y, err = readFraction(attr.Value)
					default:
						err = f.ReadGradAttr(attr)
					}
					if err != nil {
						return fmt.Errorf("error parsing linear gradient: %w", err)
					}
				}
			case "radialGradient":
				if f.Gradient == nil {
					f.Gradient = RadialGradient()
				} else {
					f.Gradient.Radial = true
				}
				var setFx, setFy bool
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					// note: id not processed here - must be done externally
					case "r":
						f.Gradient.Radius, err = readFraction(attr.Value)
					case "cx":
						f.Gradient.Center.X, err = readFraction(attr.Value)
					case "cy":
						f.Gradient.Center.Y, err = readFraction(attr.Value)
					case "fx":
						setFx = true
						f.Gradient.Focal.X, err = readFraction(attr.Value)
					case "fy":
						setFy = true
						f.Gradient.Focal.Y, err = readFraction(attr.Value)
					default:
						err = f.ReadGradAttr(attr)
					}
					if err != nil {
						return fmt.Errorf("error parsing radial gradient: %w", err)
					}
				}
				if !setFx { // set fx to cx by default
					f.Gradient.Focal.X = f.Gradient.Center.X
				}
				if !setFy { // set fy to cy by default
					f.Gradient.Focal.Y = f.Gradient.Center.Y
				}
			case "stop":
				stop := GradientStop{Opacity: 1, Color: Black}
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
						if err != nil {
							return err
						}
					case "stop-color":
						clr, err := FromString(attr.Value, nil)
						if err != nil {
							return fmt.Errorf("invalid color string: %w", err)
						}
						stop.Color = clr
					case "stop-opacity":
						var o64 float64
						o64, err = strconv.ParseFloat(attr.Value, 32)
						if err != nil {
							return err
						}
						stop.Opacity = float32(o64)
					}
					if err != nil {
						return fmt.Errorf("error parsing color stop: %w", err)
					}
				}
				if f.Gradient == nil {
					return fmt.Errorf("no gradient but stops in: %v stop: %v", f, stop)
				} else {
					f.Gradient.Stops = append(f.Gradient.Stops, stop)
				}
			default:
				return fmt.Errorf("cannot process svg element %q", se.Name.Local)
			}
		case xml.EndElement:
			if se.Name.Local == "linearGradient" || se.Name.Local == "radialGradient" {
				return nil
			}
			if se.Name.Local != "stop" {
				return fmt.Errorf("got unexpected end element: %v", se.Name.Local)
			}
		case xml.CharData:
		}
	}
	return nil
}

func readFraction(v string) (float32, error) {
	v = strings.TrimSpace(v)
	d := float32(1)
	if strings.HasSuffix(v, "%") {
		d = 100
		v = strings.TrimSuffix(v, "%")
	}
	f64, err := strconv.ParseFloat(v, 32)
	if err != nil {
		return 0, err
	}
	f := float32(f64)
	f /= d
	if f < 0 {
		f = 0
	}
	return f, nil
}

func (f *Full) ReadGradAttr(attr xml.Attr) error {
	switch attr.Name.Local {
	case "gradientTransform":
		tx := mat32.Identity2D()
		err := tx.SetString(attr.Value)
		if err != nil {
			return err
		}
		f.Gradient.Matrix = tx
	case "gradientUnits":
		switch strings.TrimSpace(attr.Value) {
		case "userSpaceOnUse":
			f.Gradient.Units = UserSpaceOnUse
		case "objectBoundingBox":
			f.Gradient.Units = ObjectBoundingBox
		}
	case "spreadMethod":
		switch strings.TrimSpace(attr.Value) {
		case "pad":
			f.Gradient.Spread = PadSpread
		case "reflect":
			f.Gradient.Spread = ReflectSpread
		case "repeat":
			f.Gradient.Spread = RepeatSpread
		}
	}
	return nil
}

// FixGradientStops applies the CSS rules to regularize the gradient stops: https://www.w3.org/TR/css3-images/#color-stop-syntax
func FixGradientStops(grad *Gradient) {
	sz := len(grad.Stops)
	if sz == 0 {
		return
	}
	splitSt := -1
	last := float32(0)
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
			per := (end - start) / float32(1+(i-splitSt))
			cur := start + per
			for j := splitSt; j < i; j++ {
				grad.Stops[j].Offset = cur
				cur += per
			}
		}
		last = st.Offset
	}
}
