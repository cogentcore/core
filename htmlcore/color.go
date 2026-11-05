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

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
)

// backgroundColor returns the color of the given background value.
func backgroundColor(v string) string {
	for _, f := range splitColors(v) {
		lf := strings.ToLower(f)
		if strings.HasPrefix(lf, "url(") || strings.Contains(lf, "gradient(") {
			continue
		}
		if lf == "transparent" || lf == "none" {
			return lf
		}
		if parseColor(f) != nil {
			return f
		}
	}
	return ""
}

// parseColor returns the given CSS color, or nil if it is not one.
func parseColor(v string) image.Image {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	if strings.EqualFold(v, "transparent") {
		return colors.Uniform(colors.Transparent)
	}
	if !strings.HasPrefix(v, "#") && !strings.Contains(v, "(") {
		// legacy colors without a #, as in bgcolor="ffffff"
		if len(v) == 6 || len(v) == 3 {
			if _, err := strconv.ParseUint(v, 16, 32); err == nil {
				v = "#" + v
			}
		}
	}
	if cl, ok := cssColorFunction(v); ok {
		return colors.Uniform(cl)
	}
	cl, err := colors.FromString(v)
	if err != nil {
		img, err := gradient.FromString(v)
		if err != nil {
			return nil
		}
		return img
	}
	return colors.Uniform(cl)
}

// cssColorFunction returns the color of the given CSS rgb(), rgba(),
// hsl() or hsla() function, with the syntax of CSS: components that
// are numbers or percentages, separated by commas or spaces, and an
// alpha (after a comma or a /) from 0 to 1 or a percentage.
func cssColorFunction(v string) (color.RGBA, bool) {
	lv := strings.ToLower(strings.TrimSpace(v))
	name, args, ok := strings.Cut(lv, "(")
	if !ok || !strings.HasSuffix(args, ")") {
		return color.RGBA{}, false
	}
	if name == "light-dark" { // the light color, as for the light color scheme
		light, _, _ := cutTopLevel(strings.TrimSuffix(args, ")"), ',')
		if cl, ok := cssColorFunction(light); ok {
			return cl, true
		}
		cl, err := colors.FromString(light)
		return color.RGBA(colors.AsRGBA(cl)), err == nil
	}
	switch name {
	case "rgb", "rgba", "hsl", "hsla", "color", "oklab", "oklch":
	default:
		return color.RGBA{}, false
	}
	args = strings.NewReplacer(",", " ", "/", " ").Replace(strings.TrimSuffix(args, ")"))
	fs := strings.Fields(args)
	space := ""
	if name == "color" && len(fs) > 0 { // color(space c1 c2 c3 [/ alpha])
		space, fs = fs[0], fs[1:]
	}
	if len(fs) < 3 {
		return color.RGBA{}, false
	}
	// num returns the component, with a percentage of the given scale
	num := func(f string, scale float64) float64 {
		f = strings.TrimSuffix(f, "deg")
		if p, ok := strings.CutSuffix(f, "%"); ok {
			n, _ := strconv.ParseFloat(p, 64)
			return n / 100 * scale
		}
		n, _ := strconv.ParseFloat(f, 64)
		return n
	}
	a := 1.0
	if len(fs) > 3 {
		a = math.Max(0, math.Min(1, num(fs[3], 1)))
	}
	var r, g, b float64
	switch name {
	case "color":
		r, g, b = num(fs[0], 1), num(fs[1], 1), num(fs[2], 1)
		switch space {
		case "srgb-linear":
			r, g, b = srgbGamma(r), srgbGamma(g), srgbGamma(b)
		case "display-p3":
			r, g, b = p3ToSRGB(r, g, b)
		} // others (as a98-rgb) are approximated as sRGB
		return rgbaFloat(r, g, b, a), true
	case "oklab", "oklch":
		l := num(fs[0], 1)
		var ca, cb float64
		if name == "oklab" {
			ca, cb = num(fs[1], 0.4), num(fs[2], 0.4)
		} else {
			c, h := num(fs[1], 0.4), num(fs[2], 360)*math.Pi/180
			ca, cb = c*math.Cos(h), c*math.Sin(h)
		}
		r, g, b = oklabToSRGB(l, ca, cb)
		return rgbaFloat(r, g, b, a), true
	}
	if name[0] == 'r' {
		r, g, b = num(fs[0], 255)/255, num(fs[1], 255)/255, num(fs[2], 255)/255
	} else {
		h := math.Mod(num(fs[0], 360), 360) / 360
		if h < 0 {
			h++
		}
		sat, l := num(fs[1], 1), num(fs[2], 1)
		if !strings.HasSuffix(fs[1], "%") {
			sat /= 100
		}
		if !strings.HasSuffix(fs[2], "%") {
			l /= 100
		}
		hue := func(p, q, t float64) float64 {
			if t < 0 {
				t++
			} else if t > 1 {
				t--
			}
			switch {
			case t < 1.0/6:
				return p + (q-p)*6*t
			case t < 0.5:
				return q
			case t < 2.0/3:
				return p + (q-p)*(2.0/3-t)*6
			}
			return p
		}
		q := l + sat - l*sat
		if l < 0.5 {
			q = l * (1 + sat)
		}
		p := 2*l - q
		r, g, b = hue(p, q, h+1.0/3), hue(p, q, h), hue(p, q, h-1.0/3)
	}
	return rgbaFloat(r, g, b, a), true
}

// rgbaFloat returns the color of the given sRGB components and alpha,
// from 0 to 1 (clipped to that range).
func rgbaFloat(r, g, b, a float64) color.RGBA {
	c8 := func(f float64) uint8 { return uint8(math.Round(math.Max(0, math.Min(1, f)) * 255)) }
	return colors.FromNRGBA(c8(r), c8(g), c8(b), c8(a))
}

// srgbLinear returns the linear value of the given sRGB component.
func srgbLinear(c float64) float64 {
	if math.Abs(c) <= 0.04045 {
		return c / 12.92
	}
	return math.Copysign(math.Pow((math.Abs(c)+0.055)/1.055, 2.4), c)
}

// srgbGamma returns the sRGB component of the given linear value.
func srgbGamma(c float64) float64 {
	if math.Abs(c) <= 0.0031308 {
		return c * 12.92
	}
	return math.Copysign(1.055*math.Pow(math.Abs(c), 1/2.4)-0.055, c)
}

// p3ToSRGB returns the sRGB components of the given display-p3 ones
// (which have the same transfer function as sRGB).
func p3ToSRGB(r, g, b float64) (float64, float64, float64) {
	r, g, b = srgbLinear(r), srgbLinear(g), srgbLinear(b)
	lr := 1.2249401*r - 0.2249404*g + 0.0000000*b
	lg := -0.0420569*r + 1.0420571*g + 0.0000000*b
	lb := -0.0196376*r - 0.0786361*g + 1.0982735*b
	return srgbGamma(lr), srgbGamma(lg), srgbGamma(lb)
}

// oklabToSRGB returns the sRGB components of the given OKLab color.
func oklabToSRGB(l, a, b float64) (float64, float64, float64) {
	l_ := l + 0.3963377774*a + 0.2158037573*b
	m_ := l - 0.1055613458*a - 0.0638541728*b
	s_ := l - 0.0894841775*a - 1.2914855480*b
	L, M, S := l_*l_*l_, m_*m_*m_, s_*s_*s_
	r := 4.0767416621*L - 3.3077115913*M + 0.2309699292*S
	g := -1.2684380046*L + 2.6097574011*M - 0.3413193965*S
	bl := -0.0041960863*L - 0.7034186147*M + 1.7076147010*S
	return srgbGamma(r), srgbGamma(g), srgbGamma(bl)
}
