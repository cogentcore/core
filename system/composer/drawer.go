// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package composer

import (
	"image"
	"image/draw"

	"cogentcore.org/core/math32"
)

const (
	// Unchanged should be used for the unchanged argument in drawer calls,
	// when the caller knows that the image is unchanged.
	Unchanged = true

	// Changed should be used for the unchanged argument to drawer calls,
	// when the image has changed since last time or its status is unknown
	Changed
)

// Drawer is an interface for image/draw style image compositing
// functionality, which is implemented for the GPU in
// [*cogentcore.org/core/gpu/gpudraw.Drawer] and in offscreen drivers.
// This is used for compositing the stack of images that together comprise
// the content of a window. It is used in [ComposerDrawer].
type Drawer interface {

	// Start starts recording a sequence of draw / fill actions,
	// which will be performed on the GPU at End().
	// This must be called prior to any Drawer operations.
	Start()

	// End ends image drawing rendering process on render target.
	End()

	// Redraw re-renders the last draw
	Redraw()

	// Copy copies the given Go source image to the render target, with the
	// same semantics as golang.org/x/image/draw.Copy, with the destination
	// implicit in the Drawer target.
	//   - Must have called Start first!
	//   - dp is the destination point.
	//   - src is the source image. If an image.Uniform, fast Fill is done.
	//   - sr is the source region, if zero full src is used; must have for Uniform.
	//   - op is the drawing operation: Src = copy source directly (blit),
	//     Over = alpha blend with existing.
	//   - unchanged should be true if caller knows that this image is unchanged
	//     from the last time it was used -- saves re-uploading to gpu.
	Copy(dp image.Point, src image.Image, sr image.Rectangle, op draw.Op, unchanged bool)

	// Scale copies the given Go source image to the render target,
	// scaling the region defined by src and sr to the destination
	// such that sr in src-space is mapped to dr in dst-space,
	// and applying an optional rotation of the source image.
	// Has the same general semantics as golang.org/x/image/draw.Scale, with the
	// destination implicit in the Drawer target.
	// If src image is an
	//   - Must have called Start first!
	//   - dr is the destination rectangle; if zero uses full dest image.
	//   - src is the source image. Uniform does not work (or make sense) here.
	//   - sr is the source region, if zero full src is used; must have for Uniform.
	//   - rotateDeg = rotation degrees to apply in the mapping:
	//     90 = left, -90 = right, 180 = invert.
	//   - op is the drawing operation: Src = copy source directly (blit),
	//     Over = alpha blend with existing.
	//   - unchanged should be true if caller knows that this image is unchanged
	//     from the last time it was used -- saves re-uploading to gpu.
	Scale(dr image.Rectangle, src image.Image, sr image.Rectangle, rotateDeg float32, op draw.Op, unchanged bool)

	// Transform copies the given Go source image to the render target,
	// with the same semantics as golang.org/x/image/draw.Transform, with the
	// destination implicit in the Drawer target.
	//   - xform is the transform mapping source to destination coordinates.
	//   - src is the source image. Uniform does not work (or make sense) here.
	//   - sr is the source region, if zero full src is used; must have for Uniform.
	//   - op is the drawing operation: Src = copy source directly (blit),
	//     Over = alpha blend with existing.
	//   - unchanged should be true if caller knows that this image is unchanged
	//     from the last time it was used -- saves re-uploading to gpu.
	Transform(xform math32.Matrix3, src image.Image, sr image.Rectangle, op draw.Op, unchanged bool)

	// Renderer is the gpu device being drawn to.
	// Could be nil on unsupported devices (such as offscreen).
	Renderer() any
}

// DrawerBase is a base implementation of [Drawer] with basic no-ops
// for most methods. Embedders need to implement DestBounds and End.
type DrawerBase struct {
	// Image is the target render image
	Image *image.RGBA
}

// Copy copies the given Go source image to the render target, with the
// same semantics as golang.org/x/image/draw.Copy, with the destination
// implicit in the Drawer target.
//   - Must have called Start first!
//   - dp is the destination point.
//   - src is the source image. If an image.Uniform, fast Fill is done.
//   - sr is the source region, if zero full src is used; must have for Uniform.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing.
//   - unchanged should be true if caller knows that this image is unchanged
//     from the last time it was used -- saves re-uploading to gpu.
func (dw *DrawerBase) Copy(dp image.Point, src image.Image, sr image.Rectangle, op draw.Op, unchanged bool) {
	draw.Draw(dw.Image, image.Rectangle{dp, dp.Add(src.Bounds().Size())}, src, sr.Min, op)
}

// Scale copies the given Go source image to the render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space,
// and applying an optional rotation of the source image.
// Has the same general semantics as golang.org/x/image/draw.Scale, with the
// destination implicit in the Drawer target.
// If src image is an
//   - Must have called Start first!
//   - dr is the destination rectangle; if zero uses full dest image.
//   - src is the source image. Uniform does not work (or make sense) here.
//   - sr is the source region, if zero full src is used; must have for Uniform.
//   - rotateDeg = rotation degrees to apply in the mapping:
//     90 = left, -90 = right, 180 = invert.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing.
//   - unchanged should be true if caller knows that this image is unchanged
//     from the last time it was used -- saves re-uploading to gpu.
func (dw *DrawerBase) Scale(dr image.Rectangle, src image.Image, sr image.Rectangle, rotateDeg float32, op draw.Op, unchanged bool) {
	// todo: use drawmatrix and x/image to implement scale.
	draw.Draw(dw.Image, dr, src, sr.Min, op)
}

// Transform copies the given Go source image to the render target,
// with the same semantics as golang.org/x/image/draw.Transform, with the
// destination implicit in the Drawer target.
//   - xform is the transform mapping source to destination coordinates.
//   - src is the source image. Uniform does not work (or make sense) here.
//   - sr is the source region, if zero full src is used; must have for Uniform.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing.
//   - unchanged should be true if caller knows that this image is unchanged
//     from the last time it was used -- saves re-uploading to gpu.
func (dw *DrawerBase) Transform(xform math32.Matrix3, src image.Image, sr image.Rectangle, op draw.Op, unchanged bool) {
	// todo: use drawmatrix and x/image to implement transform
	draw.Draw(dw.Image, sr, src, sr.Min, op)
}

// Start starts recording a sequence of draw / fill actions,
// which will be performed on the GPU at End().
// This must be called prior to any Drawer operations.
func (dw *DrawerBase) Start() {
	// no-op
}

func (dw *DrawerBase) Renderer() any {
	// no-op
	return nil
}
