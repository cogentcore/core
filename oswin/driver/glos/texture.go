// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glos

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/drawer"
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/mat32"
)

// note: use a different interface for different formats of "textures" such as a
// a depth buffer, and have ways of converting between.  Texture2D is always
// RGBA picture texture

// textureImpl manages a texture, including loading from an image file
// and activating on GPU
type textureImpl struct {
	init      bool
	handle    uint32
	name      string
	size      image.Point
	botZero   bool
	img       *image.RGBA // when loaded
	imgTmp    *image.RGBA // for grab, prior to flip
	fbuff     gpu.Framebuffer
	drawQuads gpu.BufferMgr
	fillQuads gpu.BufferMgr
	// magFilter uint32 // magnification filter
	// minFilter uint32 // minification filter
	// wrapS     uint32 // wrap mode for s coordinate
	// wrapT     uint32 // wrap mode for t coordinate
}

// Name returns the name of the texture (filename without extension
// by default)
func (tx *textureImpl) Name() string {
	return tx.name
}

// SetName sets the name of the texture
func (tx *textureImpl) SetName(name string) {
	tx.name = name
}

// Open loads texture image from file.
// format inferred from filename -- JPEG and PNG
// supported by default.
func (tx *textureImpl) Open(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	im, _, err := image.Decode(file)
	if err != nil {
		return err
	}
	return tx.SetImage(im)
}

// Image returns the current image -- typically as an *image.RGBA.
// This is the image that was last set using Open, SetImage, or GrabImage.
// Use GrabImage to get the current GPU-side image, e.g., for rendering targets.
func (tx *textureImpl) Image() image.Image {
	if !tx.init {
		if tx.img == nil {
			return nil
		}
		return tx.img
	}
	return nil
}

// GrabImage retrieves the current contents of the Texture, e.g., if it has been
// used as a rendering target (Y axis flipped so top = 0).  Returns nil if not initialized.
// Must be called with a valid gpu context and on proper thread for that context.
// Returned image points to single internal image.RGBA used for this texture --
// copy before modifying and to retain values.
func (tx *textureImpl) GrabImage() image.Image {
	if !tx.init {
		return nil
	}
	if tx.img == nil || tx.img.Bounds().Size() != tx.size {
		tx.img = image.NewRGBA(image.Rectangle{Max: tx.size})
	}
	if tx.imgTmp == nil || tx.imgTmp.Bounds().Size() != tx.size {
		tx.imgTmp = image.NewRGBA(image.Rectangle{Max: tx.size})
	}
	tx.Activate(0)

	// 0 = level = base image
	gl.GetTexImage(gl.TEXTURE_2D, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(tx.imgTmp.Pix))
	tx.ImageFlipY(tx.img, tx.imgTmp)
	return tx.img
}

// ImageFlipY flips the Y axis from a source image.RGBA into a dest.
// both must be the same size else it panics.  This utility function
// is needed for GrabImage and is made avail for general use here.
func (tx *textureImpl) ImageFlipY(dest, src *image.RGBA) {
	if dest.Rect.Size() != src.Rect.Size() {
		panic("ImageFlipY image sizes are not the same")
	}
	sz := dest.Rect.Size()
	rsz := sz.X * 4
	for y := 0; y < sz.Y; y++ {
		sy := (y - src.Rect.Min.Y) * src.Stride
		dy := (sz.Y - y - 1 - dest.Rect.Min.Y) * dest.Stride
		srow := src.Pix[sy : sy+rsz]
		drow := dest.Pix[dy : dy+rsz]
		copy(drow, srow)
	}
}

func rgbaImage(img image.Image) (*image.RGBA, error) {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba, nil
	} else {
		// Converts image to RGBA format
		rgba := image.NewRGBA(img.Bounds())
		if rgba.Stride != rgba.Rect.Size().X*4 {
			return nil, fmt.Errorf("glos Texture2D: unsupported stride")
		}
		draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)
		return rgba, nil
	}
}

// SetImage sets entire contents of the Texture from given image
// (including setting the size of the texture from that of the img).
// This is most efficiently done using an image.RGBA, but other
// formats will be converted as necessary.  Image Y axis is automatically
// flipped when transferred up to the texture, so texture has bottom = 0.
// Can be called prior to doing Activate(), in which case the image
// pixels will then initialize the GPU version of the texture.
// If called after Activate then the image is copied up to the GPU
// and texture is left in an Activate state.
// If Activate()'d, then must be called with a valid gpu context
// and on proper thread for that context.
func (tx *textureImpl) SetImage(img image.Image) error {
	rgba, err := rgbaImage(img)
	if err != nil {
		return err
	}
	tx.img = rgba
	tx.size = rgba.Rect.Size()
	if tx.init {
		tx.Transfer(0)
	}
	return nil
}

// SetSubImage uploads the sub-Image defined by src and sr to the texture.
// such that sr.Min in src-space aligns with dp in dst-space.
// The textures's contents are overwritten; the draw operator
// is implicitly draw.Src. Texture must be Activate'd to the GPU for this
// to proceed -- if Activate() has not yet been called, it will be (on texture 0).
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) SetSubImage(dp image.Point, src image.Image, sr image.Rectangle) error {
	rgba, err := rgbaImage(src)
	if err != nil {
		return err
	}

	// todo: if needed for windows, do this here:
	// buf := src.(*imageImpl)
	// buf.preUpload()

	// src2dst is added to convert from the src coordinate space to the dst
	// coordinate space. It is subtracted to convert the other way.
	src2dst := dp.Sub(sr.Min)

	// Clip to the source.
	sr = sr.Intersect(rgba.Bounds())

	// Clip to the destination.
	dr := sr.Add(src2dst)
	dr = dr.Intersect(tx.Bounds())
	if dr.Empty() {
		return nil
	}

	// Bring dr.Min in dst-space back to src-space to get the pixel image offset.
	pix := rgba.Pix[rgba.PixOffset(dr.Min.X-src2dst.X, dr.Min.Y-src2dst.Y):]

	tx.Activate(0)

	width := dr.Dx()
	if width*4 == rgba.Stride {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(dr.Min.X), int32(dr.Min.Y), int32(width), int32(dr.Dy()), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix))
		return nil
	}
	// TODO: can we use GL_UNPACK_ROW_LENGTH with glPixelStorei for stride in
	// ES 3.0, instead of uploading the pixels row-by-row?
	for y, p := dr.Min.Y, 0; y < dr.Max.Y; y++ {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(dr.Min.X), int32(y), int32(width), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix[p:]))
		p += rgba.Stride
	}
	return nil
}

// Size returns the size of the image
func (tx *textureImpl) Size() image.Point {
	return tx.size
}

func (tx *textureImpl) Bounds() image.Rectangle {
	if tx == nil {
		return image.ZR
	}
	return image.Rectangle{Max: tx.size}
}

// BotZero returns true if this texture has the Y=0 pixels at the bottom
// of the image.  Otherwise, Y=0 is at the top, which is the default
// for most images loaded from files.
func (tx *textureImpl) BotZero() bool {
	return tx.botZero
}

// SetBotZero sets whether this texture has the Y=0 pixels at the bottom
// of the image.  Otherwise, Y=0 is at the top, which is the default
// for most images loaded from files.
func (tx *textureImpl) SetBotZero(botzero bool) {
	tx.botZero = botzero
}

// SetSize sets the size of the texture.
// If texture has been Activate'd, then this resizes the GPU side as well.
// If Activate()'d, then must be called with a valid gpu context
// and on proper thread for that context.
func (tx *textureImpl) SetSize(size image.Point) {
	if tx.size == size {
		return
	}
	wasInit := tx.init
	if wasInit {
		tx.Delete()
	}
	tx.size = size
	tx.img = nil
	if wasInit {
		tx.Activate(0)
	}
}

// Activate establishes the GPU resources and handle for the
// texture, using the given texture number (0-31 range).
// If an image has already been set for this texture, then it is
// copied up to the GPU at this point -- otherwise the texture
// is nil initialized.
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) Activate(texNo int) {
	if !tx.init {
		gl.GenTextures(1, &tx.handle)
		gl.ActiveTexture(gl.TEXTURE0 + uint32(texNo))
		gl.BindTexture(gl.TEXTURE_2D, tx.handle)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		szx := int32(tx.size.X)
		szy := int32(tx.size.Y)
		if tx.img != nil {
			gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, szx, szy, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(tx.img.Pix))
		} else {
			gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, szx, szy, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(nil))
		}
		tx.init = true
	} else {
		gl.ActiveTexture(gl.TEXTURE0 + uint32(texNo))
		gl.BindTexture(gl.TEXTURE_2D, tx.handle)
	}
}

// IsActive returns true if texture has already been Activate'd
// and thus exists on the GPU
func (tx *textureImpl) IsActive() bool {
	return tx.init
}

// Handle returns the GPU handle for the texture -- only
// valid after Activate
func (tx *textureImpl) Handle() uint32 {
	return tx.handle
}

// Transfer copies current image up to the GPU, activating on given
// texture number.  Returns false if there is no image to transfer.
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) Transfer(texNo int) bool {
	if tx.img == nil {
		return false
	}
	if !tx.init {
		tx.Activate(texNo) // does transfer
		return true
	}
	tx.size = tx.img.Rect.Size()
	tx.Activate(texNo)
	szx := int32(tx.size.X)
	szy := int32(tx.size.Y)
	// note: TexImage2D automatically flips Y axis on way up to texture
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, szx, szy, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(tx.img.Pix))
	return true
}

// Delete deletes the GPU resources associated with this texture
// (requires Activate to re-establish a new one).
// Should be called prior to Go object being deleted
// (ref counting can be done externally).
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) Delete() {
	if !tx.init {
		return
	}
	if tx.drawQuads != nil {
		tx.drawQuads.Delete()
		tx.drawQuads = nil
	}
	if tx.fillQuads != nil {
		tx.fillQuads.Delete()
		tx.fillQuads = nil
	}
	tx.DeleteFramebuffer()
	gl.DeleteTextures(1, &tx.handle)
	tx.img = nil
	tx.imgTmp = nil
	tx.init = false
}

// ActivateFramebuffer creates a gpu.Framebuffer for rendering onto
// this texture (if not already created) and activates it for
// rendering.  The gpu.Texture2D interface can provide direct access
// to the created framebuffer.
// Call gpu.TheGPU.RenderToWindow() or DeActivateFramebuffer
// to return to window rendering.
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) ActivateFramebuffer() {
	tx.Activate(0)
	if tx.fbuff == nil {
		tx.fbuff = theGPU.NewFramebuffer("", tx.size, 0)
		tx.fbuff.SetTexture(tx)
	}
	tx.fbuff.Activate()
}

func (tx *textureImpl) Framebuffer() gpu.Framebuffer {
	return tx.fbuff
}

// DeActivateFramebuffer de-activates this texture's framebuffer
// for rendering (just calls gpu.TheGPU.RenderToWindow())
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) DeActivateFramebuffer() {
	theGPU.RenderToWindow()
}

// DeleteFramebuffer deletes this Texture's framebuffer
// created during ActivateFramebuffer.
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) DeleteFramebuffer() {
	if tx.fbuff != nil {
		tx.fbuff.Delete()
		tx.fbuff = nil
	}
}

// FrameDepthAt return depth (0-1) at given pixel location from texture used as a framebuffer
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) FrameDepthAt(x, y int) (float32, error) {
	if tx.fbuff == nil {
		return 0, errors.New("Texture does not have a framebuffer activated for it -- cannot read depth")
	}
	tx.ActivateFramebuffer()
	var depth float32
	gl.ReadPixels(int32(x), int32(y), 1, 1, gl.DEPTH_COMPONENT, gl.FLOAT, gl.Ptr(&depth))
	return depth, nil
}

////////////////////////////////////////////////
//   Drawer wrappers

func (tx *textureImpl) Draw(src2dst mat32.Mat3, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	sz := tx.Size()
	tx.ActivateFramebuffer()
	gpu.Draw.Viewport(tx.Bounds())
	if tx.drawQuads == nil {
		tx.drawQuads = theApp.drawQuadsBuff()
	}
	theApp.draw(sz, src2dst, src, sr, op, opts, tx.drawQuads, tx.botZero)
	tx.DeActivateFramebuffer()
}

func (tx *textureImpl) DrawUniform(src2dst mat32.Mat3, src color.Color, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	sz := tx.Size()
	tx.ActivateFramebuffer()
	gpu.Draw.Viewport(tx.Bounds())
	if tx.fillQuads == nil {
		tx.fillQuads = theApp.fillQuadsBuff()
	}
	theApp.drawUniform(sz, src2dst, src, sr, op, opts, tx.fillQuads, tx.botZero)
	tx.DeActivateFramebuffer()
}

func (tx *textureImpl) Copy(dp image.Point, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	drawer.Copy(tx, dp, src, sr, op, opts)
}

func (tx *textureImpl) Scale(dr image.Rectangle, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	drawer.Scale(tx, dr, src, sr, op, opts)
}

func (tx *textureImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	sz := tx.Size()
	tx.ActivateFramebuffer()
	gpu.Draw.Viewport(tx.Bounds())
	if tx.fillQuads == nil {
		tx.fillQuads = theApp.fillQuadsBuff()
	}
	theApp.fillRect(sz, dr, src, op, tx.fillQuads, tx.botZero)
	tx.DeActivateFramebuffer()
}
