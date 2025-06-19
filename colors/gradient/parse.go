// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// color parsing is adapted from github.com/srwiley/oksvg:
//
// Copyright 2017 The oksvg Authors. All rights reserved.
//
// created: 2/12/2017 by S.R.Wiley

package gradient

import (
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"io"
	"strconv"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"golang.org/x/net/html/charset"
)

// XMLAttr searches for given attribute in slice of xml attributes;
// returns "" if not found.
func XMLAttr(name string, attrs []xml.Attr) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

// Cache is a cache of the [image.Image] results of [FromString] calls
// for each string passed to [FromString].
var Cache map[string]image.Image

// FromString parses the given CSS image/gradient/color string and returns the resulting image.
// FromString is based on https://www.w3schools.com/css/css3_gradients.asp.
// See [UnmarshalXML] for an XML-based version. If no Context is
// provied, FromString uses [BaseContext] with [Transparent].
func FromString(str string, ctx ...colors.Context) (image.Image, error) {
	var cc colors.Context
	if len(ctx) > 0 && ctx[0] != nil {
		cc = ctx[0]
	} else {
		cc = colors.BaseContext(colors.Transparent)
	}

	if Cache == nil {
		Cache = make(map[string]image.Image)
	}
	cnm := str
	if img, ok := Cache[cnm]; ok {
		// TODO(kai): do we need to clone?
		return img, nil
	}
	str = strings.TrimSpace(str)
	if strings.HasPrefix(str, "url(") {
		img := cc.ImageByURL(str)
		if img == nil {
			return nil, fmt.Errorf("unable to find url %q", str)
		}
		return img, nil
	}
	str = strings.ToLower(str)
	if str == "none" || str == "" {
		return nil, nil
	}
	grad := "-gradient"

	gidx := strings.Index(str, grad)
	if gidx <= 0 {
		s, err := colors.FromString(str, cc.Base())
		if err != nil {
			return nil, err
		}
		return colors.Uniform(s), nil
	}

	gtyp := str[:gidx]
	rmdr := str[gidx+len(grad):]
	pidx := strings.IndexByte(rmdr, '(')
	if pidx < 0 {
		return nil, fmt.Errorf("gradient specified but parameters not found in string %q", str)
	}
	pars := rmdr[pidx+1:]
	pars = strings.TrimSuffix(pars, ");")
	pars = strings.TrimSuffix(pars, ")")

	switch gtyp {
	case "linear", "repeating-linear":
		l := NewLinear()
		if gtyp == "repeating-linear" {
			l.SetSpread(Repeat)
		}
		err := l.SetString(pars)
		if err != nil {
			return nil, err
		}
		fixGradientStops(l.Stops)
		Cache[cnm] = l
		return l, nil
	case "radial", "repeating-radial":
		r := NewRadial()
		if gtyp == "repeating-radial" {
			r.SetSpread(Repeat)
		}
		err := r.SetString(pars)
		if err != nil {
			return nil, err
		}
		fixGradientStops(r.Stops)
		Cache[cnm] = r
		return r, nil
	}
	return nil, fmt.Errorf("got unknown gradient type %q", gtyp)
}

// FromAny returns the color image specified by the given value of any type in the
// given Context. It handles values of types [color.Color], [image.Image], and string.
// If no Context is provided, it uses [BaseContext] with [Transparent].
func FromAny(val any, ctx ...colors.Context) (image.Image, error) {
	switch v := val.(type) {
	case color.Color:
		return colors.Uniform(v), nil
	case *color.Color:
		return colors.Uniform(*v), nil
	case image.Image:
		return v, nil
	case string:
		return FromString(v, ctx...)
	case *string:
		return FromString(*v, ctx...)
	}
	return nil, fmt.Errorf("gradient.FromAny: got unsupported type %T", val)
}

// gradientDegToSides maps gradient degree notation to side notation
var gradientDegToSides = map[string]string{
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

// SetString sets the linear gradient from the given CSS linear gradient string
// (only the part inside of "linear-gradient(...)") (see
// https://developer.mozilla.org/en-US/docs/Web/CSS/gradient/linear-gradient)
func (l *Linear) SetString(str string) error {
	// TODO(kai): not fully following spec yet
	plist := strings.Split(str, ", ")
	var prevColor color.Color
	stopIndex := 0
outer:
	for pidx := 0; pidx < len(plist); pidx++ {
		par := strings.TrimRight(strings.TrimSpace(plist[pidx]), ",")
		origPar := par
		switch {
		case strings.Contains(par, "deg"):
			// TODO(kai): this is not true and should be fixed to use trig
			// can't use trig, b/c need to be full 1, 0 values -- just use map
			var ok bool
			par, ok = gradientDegToSides[par]
			if !ok {
				return fmt.Errorf("invalid gradient angle %q: must be at 45 degree increments", origPar)
			}
			par = "to " + par
			fallthrough
		case strings.HasPrefix(par, "to "):
			sides := strings.Split(par[3:], " ")
			l.Start, l.End = math32.Vector2{}, math32.Vector2{}
			for _, side := range sides {
				switch side {
				case "bottom":
					l.Start.Y = 0
					l.End.Y = 1
				case "top":
					l.Start.Y = 1
					l.End.Y = 0
				case "right":
					l.Start.X = 0
					l.End.X = 1
				case "left":
					l.Start.X = 1
					l.End.X = 0
				}
			}
		case strings.HasPrefix(par, ")"):
			break outer
		default: // must be a color stop
			var stop *Stop
			if len(l.Stops) > stopIndex {
				stop = &(l.Stops[stopIndex])
			} else {
				stop = &Stop{Opacity: 1}
			}
			err := parseColorStop(stop, prevColor, par)
			if err != nil {
				return err
			}
			if len(l.Stops) <= stopIndex {
				l.Stops = append(l.Stops, *stop)
			}
			prevColor = stop.Color
			stopIndex++
		}
	}
	if len(l.Stops) > stopIndex {
		l.Stops = l.Stops[:stopIndex]
	}
	return nil
}

// SetString sets the radial gradient from the given CSS radial gradient string
// (only the part inside of "radial-gradient(...)") (see
// https://developer.mozilla.org/en-US/docs/Web/CSS/gradient/radial-gradient)
func (r *Radial) SetString(str string) error {
	// TODO(kai): not fully following spec yet
	plist := strings.Split(str, ", ")
	var prevColor color.Color
	stopIndex := 0
outer:
	for pidx := 0; pidx < len(plist); pidx++ {
		par := strings.TrimRight(strings.TrimSpace(plist[pidx]), ",")

		// currently we just ignore circle and ellipse, but we should handle them at some point
		par = strings.TrimPrefix(par, "circle")
		par = strings.TrimPrefix(par, "ellipse")
		par = strings.TrimLeft(par, " ")
		switch {
		case strings.HasPrefix(par, "at "):
			sides := strings.Split(par[3:], " ")
			for _, side := range sides {
				switch side {
				case "bottom":
					r.Center.Set(0.5, 1)
				case "top":
					r.Center.Set(0.5, 0)
				case "right":
					r.Center.Set(1, 0.5)
				case "left":
					r.Center.Set(0, 0.5)
				case "center":
					r.Center.Set(0.5, 0.5)
				}
				r.Focal = r.Center
			}
		case strings.HasPrefix(par, ")"):
			break outer
		default: // must be a color stop
			var stop *Stop
			if len(r.Stops) > stopIndex {
				stop = &r.Stops[stopIndex]
			} else {
				stop = &Stop{Opacity: 1}
			}
			err := parseColorStop(stop, prevColor, par)
			if err != nil {
				return err
			}
			if len(r.Stops) <= stopIndex {
				r.Stops = append(r.Stops, *stop)
			}
			prevColor = stop.Color
			stopIndex++
		}
	}
	if len(r.Stops) > stopIndex {
		r.Stops = r.Stops[:stopIndex]
	}
	return nil
}

// parseColorStop parses the given color stop based on the given previous color
// and parent gradient string.
func parseColorStop(stop *Stop, prev color.Color, par string) error {
	cnm := par
	if spcidx := strings.Index(par, " "); spcidx > 0 {
		cnm = par[:spcidx]
		offs := strings.TrimSpace(par[spcidx+1:])
		off, err := readFraction(offs)
		if err != nil {
			return fmt.Errorf("invalid offset %q: %w", offs, err)
		}
		stop.Pos = off
	}
	clr, err := colors.FromString(cnm, prev)
	if err != nil {
		return fmt.Errorf("got invalid color string %q: %w", cnm, err)
	}
	stop.Color = clr
	return nil
}

// NOTE: XML marshalling functionality is at [cogentcore.org/core/svg.MarshalXMLGradient] instead of here
// because it uses a lot of SVG and XML infrastructure defined there.

// ReadXML reads an XML-formatted gradient color from the given io.Reader and
// sets the properties of the given gradient accordingly.
func ReadXML(g *Gradient, reader io.Reader) error {
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
			return UnmarshalXML(g, decoder, se)
			// todo: ignore rest?
		}
	}
	return nil
}

// UnmarshalXML parses the given XML gradient color data and sets the properties
// of the given gradient accordingly.
func UnmarshalXML(g *Gradient, decoder *xml.Decoder, se xml.StartElement) error {
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
				l := NewLinear().SetEnd(math32.Vec2(1, 0)) // SVG is LTR by default

				// if we don't already have a gradient, we use this one
				if *g == nil {
					*g = l
				} else if pl, ok := (*g).(*Linear); ok {
					// if our previous gradient is also linear, we build on it
					l = pl
				}
				// fmt.Printf("lingrad %v\n", cs.Gradient)
				for _, attr := range se.Attr {
					// fmt.Printf("attr: %v val: %v\n", attr.Name.Local, attr.Value)
					switch attr.Name.Local {
					// note: id not processed here - must be done externally
					case "x1":
						l.Start.X, err = readFraction(attr.Value)
					case "y1":
						l.Start.Y, err = readFraction(attr.Value)
					case "x2":
						l.End.X, err = readFraction(attr.Value)
					case "y2":
						l.End.Y, err = readFraction(attr.Value)
					default:
						err = readGradAttr(*g, attr)
					}
					if err != nil {
						return fmt.Errorf("error parsing linear gradient: %w", err)
					}
				}
			case "radialGradient":
				r := NewRadial()

				// if we don't already have a gradient, we use this one
				if *g == nil {
					*g = r
				} else if pr, ok := (*g).(*Radial); ok {
					// if our previous gradient is also radial, we build on it
					r = pr
				}
				var setFx, setFy bool
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					// note: id not processed here - must be done externally
					case "r":
						var radius float32
						radius, err = readFraction(attr.Value)
						r.Radius.SetScalar(radius)
					case "cx":
						r.Center.X, err = readFraction(attr.Value)
					case "cy":
						r.Center.Y, err = readFraction(attr.Value)
					case "fx":
						setFx = true
						r.Focal.X, err = readFraction(attr.Value)
					case "fy":
						setFy = true
						r.Focal.Y, err = readFraction(attr.Value)
					default:
						err = readGradAttr(*g, attr)
					}
					if err != nil {
						return fmt.Errorf("error parsing radial gradient: %w", err)
					}
				}
				if !setFx { // set fx to cx by default
					r.Focal.X = r.Center.X
				}
				if !setFy { // set fy to cy by default
					r.Focal.Y = r.Center.Y
				}
			case "stop":
				stop := Stop{Color: colors.Black, Opacity: 1}
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
						stop.Pos, err = readFraction(attr.Value)
						if err != nil {
							return err
						}
					case "stop-color":
						clr, err := colors.FromString(attr.Value)
						if err != nil {
							return fmt.Errorf("invalid color string: %w", err)
						}
						stop.Color = clr
					case "stop-opacity":
						opacity, err := readFraction(attr.Value)
						if err != nil {
							return fmt.Errorf("invalid stop opacity: %w", err)
						}
						stop.Opacity = opacity
					}
				}
				if g == nil {
					return fmt.Errorf("got stop outside of gradient: %v", stop)
				}
				gb := (*g).AsBase()
				gb.Stops = append(gb.Stops, stop)

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

// readFraction reads a decimal value from the given string.
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
	return f, nil
}

// readGradAttr reads the given xml attribute onto the given gradient.
func readGradAttr(g Gradient, attr xml.Attr) error {
	gb := g.AsBase()
	switch attr.Name.Local {
	case "gradientTransform":
		err := gb.Transform.SetString(attr.Value)
		if err != nil {
			return err
		}
	case "gradientUnits":
		return gb.Units.SetString(strings.TrimSpace(attr.Value))
	case "spreadMethod":
		return gb.Spread.SetString(strings.TrimSpace(attr.Value))
	}
	return nil
}

// fixGradientStops applies the CSS rules to regularize the given gradient stops:
// https://www.w3.org/TR/css3-images/#color-stop-syntax
func fixGradientStops(stops []Stop) {
	sz := len(stops)
	if sz == 0 {
		return
	}
	splitSt := -1
	last := float32(0)
	for i := 0; i < sz; i++ {
		st := &stops[i]
		if i == sz-1 && st.Pos == 0 {
			if last < 1.0 {
				st.Pos = 1.0
			} else {
				st.Pos = last
			}
		}
		if i > 0 && st.Pos == 0 && splitSt < 0 {
			splitSt = i
			st.Pos = last
			continue
		}
		if splitSt > 0 {
			start := stops[splitSt].Pos
			end := st.Pos
			per := (end - start) / float32(1+(i-splitSt))
			cur := start + per
			for j := splitSt; j < i; j++ {
				stops[j].Pos = cur
				cur += per
			}
		}
		last = st.Pos
	}
}
