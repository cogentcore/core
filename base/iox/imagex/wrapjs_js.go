// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package imagex

import (
	"image"
	"image/color"
	"image/draw"
	"syscall/js"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/jscp"
	"cogentcore.org/core/math32"
	"github.com/oliverbestmann/webgpu/jsx"
)

// gradient matches [cogentcore.org/core/colors/gradient.Gradient] without
// importing it, which would create an import cycle in the tests of the
// packages that it depends on.
type gradient interface {
	image.Image
	Update(opacity float32, box math32.Box2, objTransform math32.Matrix2)
}

// WrapJS returns a JavaScript optimized wrapper around the given
// [image.Image] on web, and just returns the image otherwise.
func WrapJS(src image.Image) image.Image {
	if src == nil {
		return src
	}
	switch x := src.(type) {
	case *JSRGBA:
		im := &JSRGBA{}
		*im = *x
		return im
	case *image.RGBA:
		return NewJSRGBA(x, nil)
	case *image.NRGBA:
		return NewJSRGBA(x, nil)
	case *image.Uniform:
		return src
	case gradient:
		return src
	default:
		return NewJSRGBA(AsRGBA(src), nil)
	}
}

// Resize returns a resized version of the source image (which can be
// [Wrapped]), returning a [WrapJS] image handle on web and using web-native
// optimized code. Otherwise, uses medium quality Linear resize.
func Resize(src image.Image, size image.Point) image.Image {
	im := &JSRGBA{}
	options := map[string]any{"resizeWidth": size.X, "resizeHeight": size.Y, "resizeQuality": "medium"}
	switch x := src.(type) {
	case *JSRGBA:
		*im = *x
		im.JS.Bitmap = logBitmap(createImageBitmap(im.JS.Bitmap, options))
	case *image.RGBA:
		im = NewJSRGBA(x, options)
	case *image.NRGBA:
		im = NewJSRGBA(x, options)
	default:
		im = NewJSRGBA(AsRGBA(src), options)
	}
	im.bitmapOnly(size)
	return im
}

// Crop returns a cropped region of the source image (which can be
// [Wrapped]), returning a [WrapJS] image handle on web and using web-native
// optimized code. The rectangle is relative to the origin of the source
// bounds, and the result has its origin at (0, 0); any part of the
// rectangle outside of the source is transparent.
func Crop(src image.Image, rect image.Rectangle) image.Image {
	// An image decoded or resized by the browser only has its bitmap,
	// which the browser crops.
	if x, ok := src.(*JSRGBA); ok && x.RGBA == nil {
		im := &JSRGBA{}
		*im = *x
		args := []any{rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy()}
		im.JS.Bitmap = logBitmap(createImageBitmap(im.JS.Bitmap, nil, args...))
		im.bitmapOnly(rect.Size())
		return im
	}
	// Local pixels are cropped in Go, which copies only the cropped region
	// to JavaScript, and keeps the pixels and the ImageData of the result
	// consistent with its bitmap.
	var local image.Image = src
	if x, ok := src.(*JSRGBA); ok {
		local = x.RGBA
	}
	dst := image.NewRGBA(image.Rectangle{Max: rect.Size()})
	draw.Draw(dst, dst.Bounds(), local, local.Bounds().Min.Add(rect.Min), draw.Src)
	return NewJSRGBA(dst, nil)
}

// bitmapOnly sets the image to have the given size after its bitmap has
// been resized or cropped by the browser: its pixels and ImageData have the
// size of the source, so they are cleared, and its pixels are read back
// from the new bitmap if they are needed.
func (im *JSRGBA) bitmapOnly(size image.Point) {
	im.RGBA = nil
	im.JS.Data = js.Undefined()
	im.JS.Bounds = image.Rectangle{Max: size}
}

// JSRGBA is a wrapper around [image.RGBA] that adds [JSImageData] pointers.
// It implements the [Wrapped] interface.
type JSRGBA struct {
	*image.RGBA
	JS JSImageData
}

// NewJSRGBA returns a new [Wrapped] [JSRGBA] image from original source image.
// If the source is already a wrapped [JSRGBA] image, it returns a shallow
// copy of that original data, re-using the pixels and js pointers.
// Returns nil if the source image is nil.
func NewJSRGBA(src image.Image, options map[string]any) *JSRGBA {
	if src == nil {
		return nil
	}
	im := &JSRGBA{}
	switch x := src.(type) {
	case *JSRGBA:
		*im = *x
		return im
	case *image.RGBA:
		im.RGBA = x
		im.JS.SetImageData(x.Pix, src.Bounds(), options)
	case *image.NRGBA:
		if options == nil {
			options = map[string]any{}
		}
		options["premultiplyAlpha"] = "premultiply"
		im.JS.SetImageData(x.Pix, src.Bounds(), options)
		// todo: get RGBA back from web
	default:
		im.RGBA = AsRGBA(src)
		im.JS.SetImageData(im.RGBA.Pix, im.RGBA.Bounds(), options)
	}
	return im
}

// Bounds returns the underlying JS data Bounds
func (im *JSRGBA) Bounds() image.Rectangle {
	return im.JS.Bounds
}

// Update must be called any time the image has been updated!
func (im *JSRGBA) Update() {
	im.ensureRGBA()
	im.JS.SetRGBA(im.RGBA)
}

// Underlying returns the underlying image data as a local image.RGBA.
// For an image decoded by the browser, its pixels are read back from
// its ImageBitmap the first time they are needed.
func (im *JSRGBA) Underlying() image.Image {
	im.ensureRGBA()
	return im.RGBA
}

// At returns the color of the pixel at the given position,
// reading the pixels back from the ImageBitmap if needed.
func (im *JSRGBA) At(x, y int) color.Color {
	im.ensureRGBA()
	return im.RGBA.At(x, y)
}

// The other methods of image.RGBA that use its pixels are also
// overridden, so that they read the pixels back first: the draw package
// uses RGBA64At, for example.

func (im *JSRGBA) RGBAAt(x, y int) color.RGBA {
	im.ensureRGBA()
	return im.RGBA.RGBAAt(x, y)
}

func (im *JSRGBA) RGBA64At(x, y int) color.RGBA64 {
	im.ensureRGBA()
	return im.RGBA.RGBA64At(x, y)
}

func (im *JSRGBA) Opaque() bool {
	im.ensureRGBA()
	return im.RGBA.Opaque()
}

func (im *JSRGBA) SubImage(r image.Rectangle) image.Image {
	im.ensureRGBA()
	return im.RGBA.SubImage(r)
}

// Set, SetRGBA and SetRGBA64 change the pixels: [JSRGBA.Update] must be
// called after them.

func (im *JSRGBA) Set(x, y int, c color.Color) {
	im.ensureRGBA()
	im.RGBA.Set(x, y, c)
}

func (im *JSRGBA) SetRGBA(x, y int, c color.RGBA) {
	im.ensureRGBA()
	im.RGBA.SetRGBA(x, y, c)
}

func (im *JSRGBA) SetRGBA64(x, y int, c color.RGBA64) {
	im.ensureRGBA()
	im.RGBA.SetRGBA64(x, y, c)
}

// ensureRGBA makes sure that the pixels of the image are available in its
// RGBA, reading them back from its ImageBitmap if it has none, as for an
// image that the browser decoded.
func (im *JSRGBA) ensureRGBA() {
	if im.RGBA != nil || im.JS.Bitmap.IsUndefined() || im.JS.Bitmap.IsNull() {
		return
	}
	im.RGBA = readBitmapPixels(im.JS.Bitmap, im.JS.Bounds)
}

// JSImageData has JavaScript pointers to the image bytes and an ImageBitmap
// (gpu texture) of an image.
type JSImageData struct {
	// Data is the raw Pix bytes in a javascript array buffer.
	Data js.Value

	// Bitmap is the result of createImageBitmap on ImageData; a gpu texture basically.
	Bitmap js.Value

	// Bounds is the size of the image: keep updated.
	Bounds image.Rectangle
}

// SetImageData sets the JavaScript pointers from given bytes.
func (im *JSImageData) SetImageData(src []byte, sbb image.Rectangle, options map[string]any) {
	// The ImageData keeps its bytes, so they must not be a view of the
	// wasm memory, which is detached when that memory grows: then
	// createImageBitmap fails, and so would drawing the ImageData later.
	jsBuf := jscp.StableBytesToJS(src)
	imageData := js.Global().Get("ImageData").New(jsBuf, sbb.Dx(), sbb.Dy())
	im.Bounds = image.Rectangle{Max: sbb.Size()}
	im.Data = imageData
	im.Bitmap = logBitmap(createImageBitmap(imageData, options))
}

// createImageBitmap calls the JS createImageBitmap function with given args,
// returning an error with the reason if it fails, without logging it, as
// a caller can handle the failure.
func createImageBitmap(imageData js.Value, options map[string]any, args ...any) (js.Value, error) {
	var imageBitmapPromise js.Value
	if len(args) > 0 {
		args = append([]any{imageData}, args...)
		args = append(args, options)
		imageBitmapPromise = js.Global().Call("createImageBitmap", args...)
	} else {
		imageBitmapPromise = js.Global().Call("createImageBitmap", imageData, options)
	}
	imageBitmap, ok := jsx.Await(imageBitmapPromise)
	if ok {
		return imageBitmap, nil
	}
	reason := "unknown reason"
	if imageBitmap.Type() == js.TypeObject {
		reason = imageBitmap.Call("toString").String()
	}
	return js.Undefined(), errors.New("imagex.createImageBitmap failed: " + reason)
}

// logBitmap returns the given ImageBitmap, logging the error of a
// [createImageBitmap] call that the caller does not handle.
func logBitmap(bitmap js.Value, err error) js.Value {
	errors.Log(err)
	return bitmap
}

// SetRGBA sets the JavaScript pointers from given image.RGBA.
func (im *JSImageData) SetRGBA(src *image.RGBA) {
	im.SetImageData(src.Pix, src.Bounds(), nil)
}

func (im *JSImageData) SetNRGBA(src *image.NRGBA) {
	im.SetImageData(src.Pix, src.Bounds(), map[string]any{"premultiplyAlpha": "premultiply"})
}

func (im *JSImageData) Set(src image.Image) {
	if src == nil {
		return
	}
	if x, ok := src.(*JSRGBA); ok {
		*im = x.JS
		return
	}
	ui := Unwrap(src)
	switch x := ui.(type) {
	case *image.RGBA:
		im.SetRGBA(x)
	case *image.NRGBA:
		im.SetNRGBA(x)
	case *image.Uniform:
		errors.Log(errors.New("imagex.JSImageData: setting image to Uniform not supported"))
	case gradient:
		errors.Log(errors.New("imagex.JSImageData: setting image to Gradient not supported"))
	default:
		im.SetRGBA(AsRGBA(ui))
	}
}
