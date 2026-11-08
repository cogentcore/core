// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"
)

// cssFilter is the filter property of CSS for images: a list of the
// functions that transform the colors of their pixels, in order (as the
// white submenu wedges of pkg.go.dev, grey icons with a filter of
// brightness(0) saturate(1) invert(1)).
type cssFilter []filterFunc

// filterFunc is a function of a filter, with its amount.
type filterFunc struct {
	name   string
	amount float64 // a factor (1 is 100%), or degrees for hue-rotate
}

// parseFilter returns the color functions of the given filter value, or
// nil if it has none (or none that is supported).
func parseFilter(v string) cssFilter {
	var f cssFilter
	for rest := strings.TrimSpace(strings.ToLower(v)); rest != ""; {
		name, args, ok := strings.Cut(rest, "(")
		if !ok {
			break
		}
		end := strings.Index(args, ")")
		if end < 0 {
			break
		}
		arg := strings.TrimSpace(args[:end])
		rest = strings.TrimSpace(args[end+1:])
		name = strings.TrimSpace(name)
		amount := 1.0
		if name == "hue-rotate" {
			amount = 0
		}
		if arg != "" {
			a, ok := filterAmount(arg, name == "hue-rotate")
			if !ok {
				continue
			}
			amount = a
		}
		switch name {
		case "brightness", "contrast", "grayscale", "invert", "opacity", "saturate", "sepia", "hue-rotate":
			f = append(f, filterFunc{name, amount})
		}
	}
	return f
}

// filterAmount returns the amount of the given argument of a filter
// function: a number or a percentage, or an angle for hue-rotate.
func filterAmount(arg string, angle bool) (float64, bool) {
	if angle {
		for _, u := range []struct {
			suffix string
			deg    float64
		}{{"deg", 1}, {"grad", 0.9}, {"rad", 180 / math.Pi}, {"turn", 360}} {
			if n, ok := strings.CutSuffix(arg, u.suffix); ok {
				f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
				return f * u.deg, err == nil
			}
		}
		f, err := strconv.ParseFloat(arg, 64)
		return f, err == nil && f == 0
	}
	if n, ok := strings.CutSuffix(arg, "%"); ok {
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f / 100, err == nil
	}
	f, err := strconv.ParseFloat(arg, 64)
	return f, err == nil
}

// apply returns the given image with the colors of its pixels transformed
// by the filter.
func (f cssFilter) apply(img image.Image) *image.RGBA {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rectangle{Max: b.Size()})
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			dst.Set(x-b.Min.X, y-b.Min.Y, f.color(c))
		}
	}
	return dst
}

// color returns the given color transformed by the filter, as in CSS (the
// matrices of the filter effects spec, on unpremultiplied colors).
func (f cssFilter) color(c color.NRGBA) color.NRGBA {
	r, g, b, a := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, float64(c.A)/255
	matrix := func(m [9]float64) {
		r, g, b = m[0]*r+m[1]*g+m[2]*b, m[3]*r+m[4]*g+m[5]*b, m[6]*r+m[7]*g+m[8]*b
	}
	for _, fn := range f {
		v := fn.amount
		switch fn.name {
		case "brightness":
			r, g, b = r*v, g*v, b*v
		case "contrast":
			r, g, b = (r-0.5)*v+0.5, (g-0.5)*v+0.5, (b-0.5)*v+0.5
		case "invert":
			v = min(v, 1)
			r, g, b = r*(1-v)+(1-r)*v, g*(1-v)+(1-g)*v, b*(1-v)+(1-b)*v
		case "opacity":
			a *= min(v, 1)
		case "grayscale":
			v = 1 - min(v, 1)
			matrix([9]float64{0.2126 + 0.7874*v, 0.7152 - 0.7152*v, 0.0722 - 0.0722*v,
				0.2126 - 0.2126*v, 0.7152 + 0.2848*v, 0.0722 - 0.0722*v,
				0.2126 - 0.2126*v, 0.7152 - 0.7152*v, 0.0722 + 0.9278*v})
		case "sepia":
			v = 1 - min(v, 1)
			matrix([9]float64{0.393 + 0.607*v, 0.769 - 0.769*v, 0.189 - 0.189*v,
				0.349 - 0.349*v, 0.686 + 0.314*v, 0.168 - 0.168*v,
				0.272 - 0.272*v, 0.534 - 0.534*v, 0.131 + 0.869*v})
		case "saturate":
			matrix([9]float64{0.213 + 0.787*v, 0.715 - 0.715*v, 0.072 - 0.072*v,
				0.213 - 0.213*v, 0.715 + 0.285*v, 0.072 - 0.072*v,
				0.213 - 0.213*v, 0.715 - 0.715*v, 0.072 + 0.928*v})
		case "hue-rotate":
			rad := v * math.Pi / 180
			cs, sn := math.Cos(rad), math.Sin(rad)
			matrix([9]float64{0.213 + cs*0.787 - sn*0.213, 0.715 - cs*0.715 - sn*0.715, 0.072 - cs*0.072 + sn*0.928,
				0.213 - cs*0.213 + sn*0.143, 0.715 + cs*0.285 + sn*0.140, 0.072 - cs*0.072 - sn*0.283,
				0.213 - cs*0.213 - sn*0.787, 0.715 - cs*0.715 + sn*0.715, 0.072 + cs*0.928 + sn*0.072})
		}
		r, g, b = clamp01(r), clamp01(g), clamp01(b)
	}
	return color.NRGBA{uint8(r*255 + 0.5), uint8(g*255 + 0.5), uint8(b*255 + 0.5), uint8(clamp01(a)*255 + 0.5)}
}

func clamp01(v float64) float64 { return min(max(v, 0), 1) }
