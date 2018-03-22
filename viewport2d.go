// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/rcoreilly/goki/ki"
	// "golang.org/x/image/font"
	"image"
	"image/draw"
	"image/png"
	"io"
	// "log"
	"os"
	"reflect"
)

// Viewport2D provides an image and a stack of Paint contexts for drawing onto the image
// with a convenience forwarding of the Paint methods operating on the current Paint
type Viewport2D struct {
	Node2DBase
	ViewBox ViewBox2D   `svg:"viewBox",desc:"viewbox within any parent Viewport2D"`
	Render  RenderState `json:"-",desc:"render state for rendering"`
	Pixels  *image.RGBA `json:"-",desc:"pixels that we render into"`
	Backing *image.RGBA `json:"-",desc:"if non-nil, this is what goes behind our image -- copied from our region in parent image -- allows us to re-render cleanly into parent, even with transparency"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Viewport2D = ki.Types.AddType(&Viewport2D{}, nil)

// NewViewport2D creates a new image.RGBA with the specified width and height
// and prepares a context for rendering onto that image.
func NewViewport2D(width, height int) *Viewport2D {
	return NewViewport2DForRGBA(image.NewRGBA(image.Rect(0, 0, width, height)))
}

// NewViewport2DForImage copies the specified image into a new image.RGBA
// and prepares a context for rendering onto that image.
func NewViewport2DForImage(im image.Image) *Viewport2D {
	return NewViewport2DForRGBA(imageToRGBA(im))
}

// NewViewport2DForRGBA prepares a context for rendering onto the specified image.
// No copy is made.
func NewViewport2DForRGBA(im *image.RGBA) *Viewport2D {
	vp := &Viewport2D{
		ViewBox: ViewBox2D{Size: image.Point{im.Bounds().Size().X, im.Bounds().Size().Y}},
		Pixels:  im,
	}
	vp.Render.Image = vp.Pixels
	return vp
}

// resize viewport, creating a new image (no point in trying to resize the image -- need to re-render) -- updates ViewBox Size too -- triggers update -- wrap in other UpdateStart/End calls as appropriate
func (vp *Viewport2D) Resize(width, height int) {
	if vp.Pixels.Bounds().Size().X == width && vp.Pixels.Bounds().Size().Y == height {
		return // already good
	}
	vp.UpdateStart()
	vp.Pixels = image.NewRGBA(image.Rect(0, 0, width, height))
	vp.Render.Image = vp.Pixels
	vp.ViewBox.Size = image.Point{width, height}
	vp.UpdateEnd()
	vp.FullRender2DRoot()
}

////////////////////////////////////////////////////////////////////////////////////////
//  Main Rendering code

// draw our image into parents -- called at right place in Render
func (vp *Viewport2D) DrawIntoParent(parVp *Viewport2D) {
	r := vp.ViewBox.Bounds()
	if vp.Backing != nil {
		draw.Draw(parVp.Pixels, r, vp.Backing, image.ZP, draw.Src)
	}
	draw.Draw(parVp.Pixels, r, vp.Pixels, image.ZP, draw.Src)
}

// copy our backing image from parent -- called at right place in Render
func (vp *Viewport2D) CopyBacking(parVp *Viewport2D) {
	r := vp.ViewBox.Bounds()
	if vp.Backing == nil {
		vp.Backing = image.NewRGBA(vp.ViewBox.SizeRect())
	}
	draw.Draw(vp.Backing, r, parVp.Pixels, image.ZP, draw.Src)
}

func (vp *Viewport2D) DrawIntoWindow() {
	wini := vp.FindParentByType(reflect.TypeOf(Window{}))
	if wini != nil {
		win := (wini).(*Window)
		// width, height := win.Win.Size() // todo: update size of our window
		s := win.Win.Screen()
		s.CopyRGBA(vp.Pixels, vp.Pixels.Bounds())
		win.Win.FlushImage()
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Node2D interface

func (vp *Viewport2D) AsNode2D() *Node2DBase {
	return &vp.Node2DBase
}

func (vp *Viewport2D) AsViewport2D() *Viewport2D {
	return vp
}

func (vp *Viewport2D) InitNode2D() {
	vp.NodeSig.Connect(vp.This, func(vpki, vpa ki.Ki, sig int64, data interface{}) {
		vp, ok := vpki.(*Viewport2D)
		if !ok {
			return
		}
		fmt.Printf("viewport: %v rendering due to signal: %v from node: %v\n", vp.PathUnique(), sig, vpa.PathUnique())
		vp.FullRender2DRoot()
	})
}

func (vp *Viewport2D) Style2D() {
	vp.Style2DWidget()
}

func (vp *Viewport2D) Layout2D(iter int) {
	if iter == 0 {
		vp.InitLayout2D()
		vp.LayData.AllocSize.SetFromPoint(vp.ViewBox.Size)
	} else {
		vp.GeomFromLayout() // get our geom from layout -- always do this for widgets  iter > 0
		// todo: we now need to update our ViewBox based on Alloc values..
	}
}

func (vp *Viewport2D) Node2DBBox() image.Rectangle {
	return vp.ViewBox.Bounds()
}

func (vp *Viewport2D) Render2D() {
	if vp.Viewport != nil {
		vp.CopyBacking(vp.Viewport) // full re-render is when we copy the backing
		vp.DrawIntoParent(vp.Viewport)
	} else { // top-level, try drawing into window
		vp.DrawIntoWindow()
	}
}

func (vp *Viewport2D) CanReRender2D() bool {
	return true // always true for viewports
}

func (g *Viewport2D) FocusChanged2D(gotFocus bool) {
}

////////////////////////////////////////////////////////////////////////////////////////
//  Signal Handling

// each node calls this signal method to notify its parent viewport whenever it changes, causing a re-render
func SignalViewport2D(vpki, node ki.Ki, sig int64, data interface{}) {
	vpgi, ok := vpki.(Node2D)
	if !ok {
		return
	}
	vp := vpgi.AsViewport2D()
	if vp == nil { // should not happen -- should only be called on viewports
		return
	}
	gii, gi := KiToNode2D(node)
	if gii == nil { // should not happen
		return
	}
	// fmt.Printf("viewport: %v rendering due to signal: %v from node: %v\n", vp.PathUnique(), ki.SignalType(sig), node.PathUnique())

	// todo: probably need better ways of telling how much re-rendering is needed
	if sig == int64(ki.NodeSignalChildAdded) {
		vp.Init2DRoot()
		vp.Style2DRoot()
		vp.Render2DRoot()
	} else {
		if gii.CanReRender2D() {
			vp.Render2DFromNode(gi)
			vp.Render2D() // redraw us
		} else {
			vp.Style2DFromNode(gi) // restyle only from affected node downward
			vp.ReRender2DRoot()    // need to re-render entirely..
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Root-level Viewport API -- does all the recursive calls

// initialize scene graph
func (vp *Viewport2D) Init2DRoot() {
	vp.FunDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		gii, gi := KiToNode2D(k)
		if gii == nil {
			return false
		}
		gi.InitNode2DBase()
		gii.InitNode2D()
		return true
	})
}

// full render of the tree
func (vp *Viewport2D) FullRender2DRoot() {
	vp.Init2DRoot()
	vp.Style2DRoot()
	vp.Layout2DRoot()
	vp.Render2DRoot()
}

// re-render of the tree -- after it has already been initialized and styled
// -- does layout and render passes
func (vp *Viewport2D) ReRender2DRoot() {
	vp.Layout2DRoot()
	vp.Render2DRoot()
}

// do the styling -- only from root
func (vp *Viewport2D) Style2DRoot() {
	vp.Style2DFromNode(&vp.Node2DBase)
}

// do the layout pass from root
func (vp *Viewport2D) Layout2DRoot() {
	vp.Layout2DFromNode(&vp.Node2DBase)
}

// do the render from root
func (vp *Viewport2D) Render2DRoot() {
	vp.Render2DFromNode(&vp.Node2DBase)
}

// this only needs to be done on a structural update
func (vp *Viewport2D) Style2DFromNode(gn *Node2DBase) {
	gn.FunDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		gii, _ := KiToNode2D(k)
		if gii == nil {
			return false // going into a different type of thing, bail
		}
		gii.Style2D()
		return true
	})
}

// do the layout pass in 2 iterations
func (vp *Viewport2D) Layout2DFromNode(gn *Node2DBase) {
	// todo: support multiple iterations if necc

	// layout happens in depth-first manner -- requires two functions
	gn.FunDownDepthFirst(0, vp,
		func(k ki.Ki, level int, d interface{}) bool { // this is for testing whether to process node
			_, gi := KiToNode2D(k)
			if gi == nil {
				return false
			}
			if gi.Paint.Off { // off below this
				return false
			}
			return true
		},
		func(k ki.Ki, level int, d interface{}) bool {
			gii, gi := KiToNode2D(k)
			if gi == nil {
				return false
			}
			if gi.Paint.Off { // off below this
				return false
			}
			gii.Layout2D(0)
			return true
		})

	// second pass we add the parent positions after layout -- don't want to do that in
	// render b/c then it doesn't work for local re-renders..
	gn.FunDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		gii, gi := KiToNode2D(k)
		if gi == nil {
			return false
		}
		if gi.Paint.Off { // off below this
			return false
		}
		gii.Layout2D(1) // todo: check for multiple iterations needed..
		return true
	})

}

// do the render pass -- two iterations required (may need more later) --
// first does all non-viewports, and second does all viewports, which must
// come after all rendering done in them -- could add iter to method if
// viewport actually needs to be called in first render pass??
func (vp *Viewport2D) Render2DFromNode(gn *Node2DBase) {
	gn.FunDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		gii, gi := KiToNode2D(k)
		if gi == nil {
			return false
		}
		if gii.AsViewport2D() != nil { // skip viewports on first pass
			return true
		}
		if gi.Paint.Off { // off below this
			return false
		}
		gii.Render2D()
		return true
	})
	// second pass ONLY process viewports
	gn.FunDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		gii, gi := KiToNode2D(k)
		if gi == nil {
			return false
		}
		if gii.AsViewport2D() == nil { // skip NON viewports on second pass
			return true
		}
		// if gi.Paint.Off { // off below this
		// 	return false
		// }
		gii.Render2D()
		return true
	})
}

// SavePNG encodes the image as a PNG and writes it to disk.
func (vp *Viewport2D) SavePNG(path string) error {
	return SavePNG(path, vp.Pixels)
}

// EncodePNG encodes the image as a PNG and writes it to the provided io.Writer.
func (vp *Viewport2D) EncodePNG(w io.Writer) error {
	return png.Encode(w, vp.Pixels)
}

// todo:

// DrawPoint is like DrawCircle but ensures that a circle of the specified
// size is drawn regardless of the current transformation matrix. The position
// is still transformed, but not the shape of the point.
// func (vp *Viewport2D) DrawPoint(x, y, r float64) {
// 	pc := vp.PushNewPaint()
// 	p := pc.TransformPoint(x, y)
// 	pc.Identity()
// 	pc.DrawCircle(p.X, p.Y, r)
// 	vp.PopPaint()
// }

//////////////////////////////////////////////////////////////////////////////////
//  Image utilities

func LoadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	im, _, err := image.Decode(file)
	return im, err
}

func LoadPNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return png.Decode(file)
}

func SavePNG(path string, im image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, im)
}

func imageToRGBA(src image.Image) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Rect, src, image.ZP, draw.Src)
	return dst
}
