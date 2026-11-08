// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"image"
	"image/color"
	"io"
	"strconv"
	"strings"
	"sync"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/svg"
	"golang.org/x/image/draw"
)

// cssMask is the mask of an element (mask-image and its size, position,
// and repeat), as for the icons of Wikipedia, which are the background
// color of empty elements shown through the mask of an SVG image.
type cssMask struct {
	url, size, position string
	repeat              bool
	hasRepeat           bool
}

// maskSources are the data of the images of masks, by URL, for all pages,
// as they are used by many elements (as icons).
var maskSources struct {
	sync.Mutex
	data map[string]*maskSource
}

// maskSource is the image of a mask: an SVG (rasterized at the size of
// its use) or a raster image.
type maskSource struct {
	svg    []byte
	raster image.Image

	// size is the intrinsic size of the image in CSS px (0 if unknown).
	size math32.Vector2
}

// maxMaskSources is the number of mask images that are kept.
const maxMaskSources = 512

// maskSourceOf returns the image of the mask at the given URL, getting it
// once for all pages, or nil if it cannot be gotten.
func (c *Context) maskSourceOf(u string) *maskSource {
	key := u
	if !strings.HasPrefix(u, "data:") {
		if pu, err := parseRelativeURL(u, c.PageURL); err == nil {
			key = pu.String()
		}
	}
	maskSources.Lock()
	src, ok := maskSources.data[key]
	maskSources.Unlock()
	if ok {
		return src
	}
	src = c.fetchMaskSource(key)
	maskSources.Lock()
	if maskSources.data == nil || len(maskSources.data) >= maxMaskSources {
		maskSources.data = map[string]*maskSource{}
	}
	maskSources.data[key] = src
	maskSources.Unlock()
	return src
}

// fetchMaskSource gets the image of a mask at the given URL.
func (c *Context) fetchMaskSource(u string) *maskSource {
	resp, err := Get(c, u)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil
	}
	if strings.Contains(resp.Header.Get("Content-Type"), "svg") || looksLikeSVG(data) {
		src := &maskSource{svg: data}
		sv := svg.NewSVG(math32.Vec2(16, 16))
		if sv.ReadXML(bytes.NewReader(data)) == nil {
			src.size = math32.Vec2(sv.PhysicalWidth.Dots, sv.PhysicalHeight.Dots)
			if src.size.X <= 0 || src.size.Y <= 0 {
				src.size = sv.Root.ViewBox.Size
			}
		}
		return src
	}
	img, _, err := imagex.Read(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return &maskSource{raster: img, size: math32.FromPoint(img.Bounds().Size())}
}

// maskFill is the background of an element with a mask: its color where
// the mask image (placed in the box of the element by the size, position
// and repeat of the mask) is opaque. It is a [gradient.Gradient] so that
// it is updated with the box of the element before it is rendered.
type maskFill struct {
	base  gradient.Base
	color color.RGBA
	src   *maskSource
	mask  cssMask

	// dp is the number of dots per CSS px.
	dp float32

	mu      sync.Mutex
	placed  image.Rectangle // the tile of the mask in the pixels of the box
	box     image.Rectangle
	tile    *image.Alpha
	opacity float32

	// inverse is the inverse of the rotation of the rendering (as of a
	// caret with a rotate transform), if any, around the center of the
	// box, with which pixels are mapped to the mask.
	inverse   math32.Matrix2
	isRotated bool
}

func (mf *maskFill) AsBase() *gradient.Base { return &mf.base }

// CopyGradient returns a copy of the mask fill (as for its state colors),
// which places its mask again.
func (mf *maskFill) CopyGradient() gradient.Gradient {
	cp := &maskFill{color: mf.color, src: mf.src, mask: mf.mask, dp: mf.dp}
	cp.base.ApplyFuncs = mf.base.ApplyFuncs.Clone()
	return cp
}

func (mf *maskFill) ColorModel() color.Model { return color.RGBAModel }

func (mf *maskFill) Bounds() image.Rectangle { return image.Rect(-1e9, -1e9, 1e9, 1e9) }

// Update places the mask in the given box (in pixels), rendering its image
// at the size that it is placed at.
func (mf *maskFill) Update(opacity float32, box math32.Box2, objTransform math32.Matrix2) {
	mf.mu.Lock()
	defer mf.mu.Unlock()
	mf.opacity = opacity
	// the rotation (and skew) of the transform, without its translation
	// and scale, which the box already has
	rot := objTransform
	rot.X0, rot.Y0 = 0, 0
	sx, sy := rot.ExtractScale()
	mf.isRotated = rot.XY != 0 || rot.YX != 0
	if mf.isRotated && sx != 0 && sy != 0 {
		c := box.Center()
		rot = rot.Scale(1/sx, 1/sy)
		mf.inverse = math32.Translate2D(c.X, c.Y).Mul(rot.Inverse()).Mul(math32.Translate2D(-c.X, -c.Y))
	}
	b := box.ToRect()
	if b == mf.box && mf.tile != nil {
		return
	}
	mf.box = b
	bsz := math32.FromPoint(b.Size())
	isz := mf.src.size.MulScalar(mf.dp)
	if isz.X <= 0 || isz.Y <= 0 {
		isz = bsz
	}
	sz := maskSize(mf.mask.size, bsz, isz, mf.dp)
	px, py := maskPosition(mf.mask.position)
	pos := math32.Vec2(px*(bsz.X-sz.X), py*(bsz.Y-sz.Y))
	mf.placed = image.Rect(0, 0, int(math32.Round(sz.X)), int(math32.Round(sz.Y))).Add(b.Min).Add(image.Pt(int(math32.Round(pos.X)), int(math32.Round(pos.Y))))
	mf.tile = mf.render(mf.placed.Size())
}

// render renders the mask image at the given size, as its alpha.
func (mf *maskFill) render(size image.Point) *image.Alpha {
	if size.X <= 0 || size.Y <= 0 || size.X > 4096 || size.Y > 4096 {
		return nil
	}
	var img image.Image
	if mf.src.svg != nil {
		sv := svg.NewSVG(math32.FromPoint(size))
		if sv.ReadXML(bytes.NewReader(mf.src.svg)) != nil {
			return nil
		}
		img = sv.RenderImage()
	} else {
		dst := image.NewRGBA(image.Rectangle{Max: size})
		draw.CatmullRom.Scale(dst, dst.Bounds(), mf.src.raster, mf.src.raster.Bounds(), draw.Src, nil)
		img = dst
	}
	a := image.NewAlpha(image.Rectangle{Max: size})
	draw.Draw(a, a.Bounds(), img, img.Bounds().Min, draw.Src)
	return a
}

// At returns the color of the background at the given pixel: its color,
// with the alpha of the mask there.
func (mf *maskFill) At(x, y int) color.Color {
	mf.mu.Lock()
	defer mf.mu.Unlock()
	if mf.tile == nil {
		return color.RGBA{}
	}
	pt := image.Pt(x, y)
	if mf.isRotated {
		q := mf.inverse.MulPoint(math32.Vec2(float32(x)+0.5, float32(y)+0.5))
		pt = image.Pt(int(math32.Floor(q.X)), int(math32.Floor(q.Y)))
	}
	p := pt.Sub(mf.placed.Min)
	ts := mf.placed.Size()
	if mf.mask.repeat && ts.X > 0 && ts.Y > 0 {
		p.X, p.Y = ((p.X%ts.X)+ts.X)%ts.X, ((p.Y%ts.Y)+ts.Y)%ts.Y
	} else if !p.In(image.Rectangle{Max: ts}) {
		return color.RGBA{}
	}
	a := float32(mf.tile.AlphaAt(p.X, p.Y).A) / 255 * mf.opacity
	c := mf.color
	res := color.Color(color.RGBA{uint8(float32(c.R) * a), uint8(float32(c.G) * a), uint8(float32(c.B) * a), uint8(float32(c.A) * a)})
	if len(mf.base.ApplyFuncs) > 0 {
		res = mf.base.ApplyFuncs.Apply(res)
	}
	return res
}

// maskSize returns the size of the mask image in the given box, for the
// given mask-size (as "20px auto", "contain", "cover", or "50%"), and the
// intrinsic size of the image, in pixels, with the given dots per px.
func maskSize(v string, box, intrinsic math32.Vector2, dp float32) math32.Vector2 {
	aspect := intrinsic.Y / intrinsic.X
	switch strings.TrimSpace(v) {
	case "contain", "cover":
		sx, sy := box.X/intrinsic.X, box.Y/intrinsic.Y
		s := min(sx, sy)
		if v == "cover" {
			s = max(sx, sy)
		}
		return intrinsic.MulScalar(s)
	case "", "auto", "auto auto":
		return intrinsic
	}
	fs := strings.Fields(v)
	length := func(f string, ref float32) (float32, bool) {
		if pct, ok := strings.CutSuffix(f, "%"); ok {
			n, err := strconv.ParseFloat(pct, 32)
			return float32(n) / 100 * ref, err == nil
		}
		if f == "auto" {
			return 0, false
		}
		n, err := strconv.ParseFloat(strings.TrimSuffix(f, "px"), 32)
		return float32(n) * dp, err == nil
	}
	w, wok := length(fs[0], box.X)
	h, hok := float32(0), false
	if len(fs) > 1 {
		h, hok = length(fs[1], box.Y)
	}
	switch {
	case wok && hok:
		return math32.Vec2(w, h)
	case wok:
		return math32.Vec2(w, w*aspect)
	case hok:
		return math32.Vec2(h/aspect, h)
	}
	return intrinsic
}

// maskPosition returns the fractions of the extra space of the box on each
// axis that are before the mask image, for the given mask-position (as
// "50% 50%", "center", or "left top").
func maskPosition(v string) (x, y float32) {
	fs := strings.Fields(strings.ToLower(v))
	if len(fs) == 0 {
		return 0, 0
	}
	if len(fs) == 1 {
		fs = append(fs, "center")
	}
	frac := func(f string) float32 {
		switch f {
		case "left", "top":
			return 0
		case "center":
			return 0.5
		case "right", "bottom":
			return 1
		}
		if pct, ok := strings.CutSuffix(f, "%"); ok {
			n, _ := strconv.ParseFloat(pct, 32)
			return float32(n) / 100
		}
		return 0
	}
	if fs[0] == "top" || fs[0] == "bottom" || fs[1] == "left" || fs[1] == "right" {
		fs[0], fs[1] = fs[1], fs[0] // as "top left"
	}
	return frac(fs[0]), frac(fs[1])
}

// maskURL returns the URL of the url() of the given value of mask-image
// (or the mask shorthand), or "" if it has none (as none).
func maskURL(v string) string {
	i := strings.Index(strings.ToLower(v), "url(")
	if i < 0 {
		return ""
	}
	rest := v[i+4:]
	end := strings.Index(rest, ")")
	if t := strings.TrimSpace(rest); strings.HasPrefix(t, `"`) || strings.HasPrefix(t, `'`) {
		// a CSS string, with backslash escapes (as the quotes of the
		// attributes of an SVG in a data URL)
		q := t[0]
		var sb strings.Builder
		for i := 1; i < len(t); i++ {
			switch c := t[i]; {
			case c == '\\' && i+1 < len(t):
				i++
				sb.WriteByte(t[i])
			case c == q:
				return sb.String()
			default:
				sb.WriteByte(c)
			}
		}
		return ""
	}
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}

// applyMask sets the background of the given style to its color through
// the mask of the given state, if it has one and a color.
func (c *Context) applyMask(st *cssState, s *styles.Style) {
	m := st.mask
	if m == nil || m.url == "" {
		return
	}
	u, ok := s.Background.(*image.Uniform)
	if !ok {
		return
	}
	src := c.maskSourceOf(m.url)
	if src == nil {
		s.Background = nil // the mask hides everything
		return
	}
	dp := s.UnitContext.Dots(units.UnitDp)
	if dp <= 0 {
		dp = 1
	}
	mk := *m
	if !mk.hasRepeat { // the mask repeats by default, as in CSS
		mk.repeat = true
	}
	s.Background = &maskFill{color: colors.AsRGBA(u.C), src: src, mask: mk, dp: dp}
}

// compileMask compiles the given mask property (mask-image, mask-size,
// mask-position, mask-repeat, or the mask shorthand, also with the -webkit-
// prefix), or returns nil if it is not one.
func compileMask(prop, val string) styleOp {
	prop = strings.TrimPrefix(prop, "-webkit-")
	lval := strings.ToLower(strings.TrimSpace(val))
	set := func(f func(m *cssMask)) styleOp {
		return func(st *cssState, s, ps *styles.Style) {
			if st.mask == nil {
				st.mask = &cssMask{}
			}
			f(st.mask)
		}
	}
	switch prop {
	case "mask-image":
		u := maskURL(val)
		return set(func(m *cssMask) { m.url = u })
	case "mask-size":
		return func(st *cssState, s, ps *styles.Style) {
			if st.mask == nil {
				st.mask = &cssMask{}
			}
			st.mask.size = resolveMaskSize(st.env, lval)
		}
	case "mask-position":
		return set(func(m *cssMask) { m.position = lval })
	case "mask-repeat":
		return set(func(m *cssMask) { m.repeat, m.hasRepeat = !strings.Contains(lval, "no-repeat"), true })
	case "mask":
		u := maskURL(val)
		rest := lval
		if i := strings.Index(rest, "url("); i >= 0 {
			if j := strings.Index(rest[i:], ")"); j >= 0 {
				rest = rest[:i] + rest[i+j+1:]
			}
		}
		pos, size, _ := strings.Cut(rest, "/")
		var keep []string
		repeat, hasRepeat := false, false
		for _, f := range strings.Fields(pos) {
			switch f {
			case "no-repeat":
				hasRepeat = true
			case "repeat", "repeat-x", "repeat-y", "round", "space":
				repeat, hasRepeat = true, true
			case "alpha", "luminance", "match-source", "add", "subtract", "intersect", "exclude",
				"border-box", "padding-box", "content-box", "fill-box", "stroke-box", "view-box", "no-clip":
			default:
				keep = append(keep, f)
			}
		}
		return func(st *cssState, s, ps *styles.Style) {
			st.mask = &cssMask{url: u, position: strings.Join(keep, " "), size: resolveMaskSize(st.env, strings.TrimSpace(size)), repeat: repeat, hasRepeat: hasRepeat}
		}
	}
	return nil
}

// resolveMaskSize returns the given mask-size with its lengths (as em and
// calc()) resolved to px with the given length environment, as maskSize
// reads px, percentages, auto, contain, and cover.
func resolveMaskSize(env *lengthEnv, v string) string {
	parts := splitTopLevelSpaces(v)
	for i, p := range parts {
		switch {
		case p == "auto" || p == "contain" || p == "cover" || strings.HasSuffix(p, "%"):
			continue
		}
		if l, ok := env.length(p, units.UnitPw); ok && l.Unit == units.UnitDp { // (% are of the box)
			parts[i] = strconv.FormatFloat(float64(l.Value), 'f', -1, 32) + "px"
		}
	}
	return strings.Join(parts, " ")
}

// splitTopLevelSpaces splits the given value at the spaces that are not in
// parentheses (as "calc(1rem + 4px) auto").
func splitTopLevelSpaces(v string) []string {
	var parts []string
	depth, start := 0, -1
	for i, r := range v {
		switch {
		case r == '(':
			depth++
		case r == ')':
			depth--
		case r == ' ' && depth == 0:
			if start >= 0 {
				parts = append(parts, v[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		parts = append(parts, v[start:])
	}
	return parts
}
