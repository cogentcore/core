package scanx

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/BurntSushi/xgbutil/xgraphics"
	"github.com/srwiley/rasterx"
)

const (
	m         = 1<<16 - 1
	mp        = 0x100 * m
	pa uint32 = 0x101
	q  uint32 = 0xFF00
)

type (
	spanCell struct {
		x0, x1, next int
		clr          color.RGBA
	}

	baseSpanner struct {
		// drawing is done with bounds.Min as the origin
		bounds image.Rectangle
		// Op is how pixels are overlayed
		Op      draw.Op
		fgColor color.RGBA
	}

	// LinkListSpanner is a Spanner that draws Spans onto a draw.Image
	// interface satisfying struct but it is optimized for *xgraphics.Image
	// and *image.RGBA image types
	// It uses a solid Color only for fg and bg and does not support a color function
	// used by gradients. Spans are accumulated into a set of linked lists, one for
	// every horizontal line in the image. After the spans for the image are accumulated,
	// use the DrawToImage function to write the spans to an image.
	LinkListSpanner struct {
		baseSpanner
		spans        []spanCell
		bgColor      color.RGBA
		lastY, lastP int
	}

	// ImgSpanner is a Spanner that draws Spans onto *xgraphics.Image
	// or *image.RGBA image types
	// It uses either a color function as a the color source, or a fgColor
	// if colFunc is nil.
	ImgSpanner struct {
		baseSpanner
		pix    []uint8
		stride int

		// xgraphics.Images swap r and b pixel values
		// compared to saved rgb value.
		xpixel    bool
		colorFunc rasterx.ColorFunc
	}
)

//Clear clears the current spans
func (x *LinkListSpanner) Clear() {
	x.lastY, x.lastP = 0, 0
	x.spans = x.spans[0:0]
	width := x.bounds.Dy()
	for i := 0; i < width; i++ {
		// The first cells are indexed according to the y values
		// to create y separate linked lists corresponding to the
		// image y length. Since index 0 is used by the first of these sentinel cells
		// 0 can and is used for the end of list value by the spanner linked list.
		x.spans = append(x.spans, spanCell{})
	}
}

func (x *LinkListSpanner) spansToImage(img draw.Image) {
	for y := 0; y < x.bounds.Dy(); y++ {
		p := x.spans[y].next
		for p != 0 {
			spCell := x.spans[p]
			clr := spCell.clr
			x0, x1 := spCell.x0, spCell.x1
			for x := x0; x < x1; x++ {
				img.Set(y, x, clr)
			}
			p = spCell.next
		}
	}
}

func (x *LinkListSpanner) spansToPix(pix []uint8, stride int, xpixel bool) {
	for y := 0; y < x.bounds.Dy(); y++ {
		yo := y * stride
		p := x.spans[y].next
		for p != 0 {
			spCell := x.spans[p]
			i0 := yo + spCell.x0*4
			i1 := i0 + (spCell.x1-spCell.x0)*4
			r, g, b, a := spCell.clr.R, spCell.clr.G, spCell.clr.B, spCell.clr.A
			if xpixel { // R and B are reversed in xgraphics.Image vs image.RGBA
				r, b = b, r
			}
			for i := i0; i < i1; i += 4 {
				pix[i+0] = r
				pix[i+1] = g
				pix[i+2] = b
				pix[i+3] = a
			}
			p = spCell.next
		}
	}
}

//DrawToImage draws the accumulated y spans onto the img
func (x *LinkListSpanner) DrawToImage(img image.Image) {
	switch img := img.(type) {
	case *xgraphics.Image:
		x.spansToPix(img.Pix, img.Stride, true)
	case *image.RGBA:
		x.spansToPix(img.Pix, img.Stride, false)
	case draw.Image:
		x.spansToImage(img)
	}
}

// SetBounds sets the spanner boundaries
func (x *LinkListSpanner) SetBounds(bounds image.Rectangle) {
	x.bounds = bounds
	x.Clear()
}

func getColorRGBA(c interface{}) (rgba color.RGBA) {
	switch c := c.(type) {
	case color.Color:
		r, g, b, a := c.RGBA()
		rgba = color.RGBA{
			R: uint8(r >> 8),
			G: uint8(g >> 8),
			B: uint8(b >> 8),
			A: uint8(a >> 8)}
	}
	return
}

func (x *LinkListSpanner) blendColor(under color.RGBA, ma uint32) color.RGBA {
	if ma == 0 {
		return under
	}
	rma := uint32(x.fgColor.R) * ma
	gma := uint32(x.fgColor.G) * ma
	bma := uint32(x.fgColor.B) * ma
	ama := uint32(x.fgColor.A) * ma
	if x.Op != draw.Over || under.A == 0 || ama == m*0xFF {
		return color.RGBA{
			uint8(rma / q),
			uint8(gma / q),
			uint8(bma / q),
			uint8(ama / q)}
	}
	a := m - (ama / (m >> 8))
	cc := color.RGBA{
		uint8((uint32(under.R)*a + rma) / q),
		uint8((uint32(under.G)*a + gma) / q),
		uint8((uint32(under.B)*a + bma) / q),
		uint8((uint32(under.A)*a + ama) / q)}
	return cc
}

func (x *LinkListSpanner) addLink(x0, x1, next, pp int, underColor color.RGBA, alpha uint32) (p int) {
	clr := x.blendColor(underColor, alpha)
	if pp >= x.bounds.Dy() && x.spans[pp].x1 >= x0 && ((clr.A == 0 && x.spans[pp].clr.A == 0) || clr == x.spans[pp].clr) {
		// Just extend the prev span; a new one is not required
		x.spans[pp].x1 = x1
		return pp
	}
	x.spans = append(x.spans, spanCell{x0: x0, x1: x1, next: next, clr: clr})
	p = len(x.spans) - 1
	x.spans[pp].next = p
	return
}

// GetSpanFunc returns the function that consumes a span described by the parameters.
func (x *LinkListSpanner) GetSpanFunc() SpanFunc {
	x.lastY = -1 // x within a y list may no longer be ordered, so this ensures a reset.
	return x.SpanOver
}

// SpanOver adds the span into an array of linked lists of spans using the fgColor and Porter-Duff composition
// ma is the accumulated alpha coverage. This function also assumes usage sorted x inputs for each y and so if
// inputs for x in y are not monotonically increasing, then lastY should be set to -1.
func (x *LinkListSpanner) SpanOver(yi, xi0, xi1 int, ma uint32) {
	if yi != x.lastY { // If the y place has changed, start at the list beginning
		x.lastP = yi
		x.lastY = yi
	}
	// since spans are sorted, we can start from x.lastP
	pp := x.lastP
	p := x.spans[pp].next
	for p != 0 && xi0 < xi1 {
		sp := x.spans[p]
		if sp.x1 <= xi0 { //sp is before new span
			pp = p
			p = sp.next
			continue
		}
		if sp.x0 >= xi1 { //new span is before sp
			x.lastP = x.addLink(xi0, xi1, p, pp, x.bgColor, ma)
			return
		}
		// left span
		if xi0 < sp.x0 {
			pp = x.addLink(xi0, sp.x0, p, pp, x.bgColor, ma)
			xi0 = sp.x0
		} else if xi0 > sp.x0 {
			pp = x.addLink(sp.x0, xi0, p, pp, sp.clr, 0)
		}

		clr := x.blendColor(sp.clr, ma)
		sameClrs := pp >= x.bounds.Dy() && ((clr.A == 0 && x.spans[pp].clr.A == 0) || clr == x.spans[pp].clr)
		if xi1 < sp.x1 { // span does not go beyond sp
			// merge with left span
			if x.spans[pp].x1 >= xi0 && sameClrs {
				x.spans[pp].x1 = xi1
				x.spans[pp].next = sp.next
				// Suffices not to advance lastP ?!? Testing says NO!
				x.lastP = yi // We need to go back, so let's just go to start of the list next time
				p = pp
			} else {
				// middle span; replaces sp
				x.spans[p] = spanCell{x0: xi0, x1: xi1, next: sp.next, clr: clr}
				x.lastP = pp
			}
			x.addLink(xi1, sp.x1, sp.next, p, sp.clr, 0)
			return
		}
		if x.spans[pp].x1 >= xi0 && sameClrs { // Extend and merge with previous
			x.spans[pp].x1 = sp.x1
			x.spans[pp].next = sp.next
			p = sp.next // clip out the current span from the list
			xi0 = sp.x1 // set remaining to start for next loop
			continue
		}
		// Set current span to start of new span and combined color
		x.spans[p] = spanCell{x0: xi0, x1: sp.x1, next: sp.next, clr: clr}
		xi0 = sp.x1 // any remaining span starts at sp.x1
		pp = p
		p = sp.next
	}
	x.lastP = pp
	if xi0 < xi1 { // add any remaining span to the end of the chain
		x.addLink(xi0, xi1, 0, pp, x.bgColor, ma)
	}
}

// SetBgColor sets the background color for blending
func (x *LinkListSpanner) SetBgColor(c interface{}) {
	x.bgColor = getColorRGBA(c)
}

// SetColor sets the color of x if it is a color.Color and ignores a rasterx.ColorFunction
func (x *LinkListSpanner) SetColor(c interface{}) {
	x.fgColor = getColorRGBA(c)
}

// NewImgSpanner returns an ImgSpanner set to draw to the img.
// Img argument must be a *xgraphics.Image or *image.Image type
func NewImgSpanner(img interface{}) (x *ImgSpanner) {
	x = &ImgSpanner{}
	x.SetImage(img)
	return
}

//SetImage set the image that the XSpanner will draw onto
func (x *ImgSpanner) SetImage(img interface{}) {
	switch img := img.(type) {
	case *xgraphics.Image:
		x.pix = img.Pix
		x.stride = img.Stride
		x.xpixel = true
		x.bounds = img.Bounds()
	case *image.RGBA:
		x.pix = img.Pix
		x.stride = img.Stride
		x.xpixel = false
		x.bounds = img.Bounds()
	}
}

// SetColor sets the color of x to either a color.Color or a rasterx.ColorFunction
func (x *ImgSpanner) SetColor(c interface{}) {
	switch c := c.(type) {
	case color.Color:
		x.colorFunc = nil
		r, g, b, a := c.RGBA()
		if x.xpixel == true { // apparently r and b values swap in xgraphics.Image
			r, b = b, r
		}
		x.fgColor = color.RGBA{
			R: uint8(r >> 8),
			G: uint8(g >> 8),
			B: uint8(b >> 8),
			A: uint8(a >> 8)}
	case rasterx.ColorFunc:
		x.colorFunc = c
	}
}

// GetSpanFunc returns the function that consumes a span described by the parameters.
// The next four func declarations are all slightly different
// but in order to reduce code redundancy, this method is used
// to dispatch the function in the draw method.
func (x *ImgSpanner) GetSpanFunc() SpanFunc {
	var (
		useColorFunc = x.colorFunc != nil
		drawOver     = x.Op == draw.Over
	)
	switch {
	case useColorFunc && drawOver:
		return x.SpanColorFunc
	case useColorFunc && !drawOver:
		return x.SpanColorFuncR
	case !useColorFunc && !drawOver:
		return x.SpanFgColorR
	default:
		return x.SpanFgColor
	}
}

//SpanColorFuncR draw the span using a colorFunc and replaces the previous values.
func (x *ImgSpanner) SpanColorFuncR(yi, xi0, xi1 int, ma uint32) {
	i0 := (yi)*x.stride + (xi0)*4
	i1 := i0 + (xi1-xi0)*4
	cx := xi0
	for i := i0; i < i1; i += 4 {
		rcr, rcg, rcb, rca := x.colorFunc(cx, yi).RGBA()
		if x.xpixel == true {
			rcr, rcb = rcb, rcr
		}
		cx++
		x.pix[i+0] = uint8(rcr * ma / mp)
		x.pix[i+1] = uint8(rcg * ma / mp)
		x.pix[i+2] = uint8(rcb * ma / mp)
		x.pix[i+3] = uint8(rca * ma / mp)
	}
}

//SpanFgColorR draws the span with the fore ground color and replaces the previous values.
func (x *ImgSpanner) SpanFgColorR(yi, xi0, xi1 int, ma uint32) {
	i0 := (yi)*x.stride + (xi0)*4
	i1 := i0 + (xi1-xi0)*4
	cr, cg, cb, ca := x.fgColor.RGBA()
	rma := uint8(cr * ma / mp)
	gma := uint8(cg * ma / mp)
	bma := uint8(cb * ma / mp)
	ama := uint8(ca * ma / mp)
	for i := i0; i < i1; i += 4 {
		x.pix[i+0] = rma
		x.pix[i+1] = gma
		x.pix[i+2] = bma
		x.pix[i+3] = ama
	}
}

//SpanColorFunc draws the span using a colorFunc and the  Porter-Duff composition operator.
func (x *ImgSpanner) SpanColorFunc(yi, xi0, xi1 int, ma uint32) {
	i0 := (yi)*x.stride + (xi0)*4
	i1 := i0 + (xi1-xi0)*4
	cx := xi0

	for i := i0; i < i1; i += 4 {
		// uses the Porter-Duff composition operator.
		rcr, rcg, rcb, rca := x.colorFunc(cx, yi).RGBA()
		if x.xpixel == true {
			rcr, rcb = rcb, rcr
		}
		cx++
		a := (m - (rca * ma / m)) * pa
		dr := uint32(x.pix[i+0])
		dg := uint32(x.pix[i+1])
		db := uint32(x.pix[i+2])
		da := uint32(x.pix[i+3])
		x.pix[i+0] = uint8((dr*a + rcr*ma) / mp)
		x.pix[i+1] = uint8((dg*a + rcg*ma) / mp)
		x.pix[i+2] = uint8((db*a + rcb*ma) / mp)
		x.pix[i+3] = uint8((da*a + rca*ma) / mp)
	}
}

//SpanFgColor draw the span using the fore ground color and the Porter-Duff composition operator.
func (x *ImgSpanner) SpanFgColor(yi, xi0, xi1 int, ma uint32) {
	i0 := (yi)*x.stride + (xi0)*4
	i1 := i0 + (xi1-xi0)*4
	// uses the Porter-Duff composition operator.
	cr, cg, cb, ca := x.fgColor.RGBA()
	ama := ca * ma
	if ama == 0xFFFF*0xFFFF { // undercolor is ignored
		rmb := uint8(cr * ma / mp)
		gmb := uint8(cg * ma / mp)
		bmb := uint8(cb * ma / mp)
		amb := uint8(ama / mp)
		for i := i0; i < i1; i += 4 {
			x.pix[i+0] = rmb
			x.pix[i+1] = gmb
			x.pix[i+2] = bmb
			x.pix[i+3] = amb
		}
		return
	}
	rma := cr * ma
	gma := cg * ma
	bma := cb * ma
	a := (m - (ama / m)) * pa
	for i := i0; i < i1; i += 4 {
		x.pix[i+0] = uint8((uint32(x.pix[i+0])*a + rma) / mp)
		x.pix[i+1] = uint8((uint32(x.pix[i+1])*a + gma) / mp)
		x.pix[i+2] = uint8((uint32(x.pix[i+2])*a + bma) / mp)
		x.pix[i+3] = uint8((uint32(x.pix[i+3])*a + ama) / mp)
	}
}
