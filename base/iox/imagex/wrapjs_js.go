// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package imagex

import (
	"image"
	"syscall/js"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors/gradient"
	"github.com/cogentcore/webgpu/jsx"
)

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
	case gradient.Gradient:
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
		im.JS.Bitmap, _ = createImageBitmap(im.JS.Bitmap, options)
	case *image.RGBA:
		im = NewJSRGBA(x, options)
	case *image.NRGBA:
		im = NewJSRGBA(x, options)
	default:
		im = NewJSRGBA(AsRGBA(src), options)
	}
	im.JS.Bounds = image.Rectangle{Max: size}
	return im
}

// Crop returns a cropped region of the source image (which can be
// [Wrapped]), returning a [WrapJS] image handle on web and using web-native
// optimized code.
func Crop(src image.Image, rect image.Rectangle) image.Image {
	im := &JSRGBA{}
	args := []any{rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy()}
	switch x := src.(type) {
	case *JSRGBA:
		*im = *x
		im.JS.Bitmap, _ = createImageBitmap(im.JS.Bitmap, nil, args...)
		return im
	case *image.RGBA:
		im = NewJSRGBA(x, nil)
	case *image.NRGBA:
		im = NewJSRGBA(x, nil)
	default:
		im = NewJSRGBA(AsRGBA(src), nil)
	}
	im.JS.Bitmap, _ = createImageBitmap(im.JS.Bitmap, nil, args...)
	im.JS.Bounds = image.Rectangle{Max: rect.Size()}
	return im
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
	im.JS.SetRGBA(im.RGBA)
}

func (im *JSRGBA) Underlying() image.Image {
	return im.RGBA
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

// setImageData sets the JavaScript pointers from given bytes.
func (im *JSImageData) SetImageData(src []byte, sbb image.Rectangle, options map[string]any) {
	jsBuf := jsx.BytesToJS(src)
	imageData := js.Global().Get("ImageData").New(jsBuf, sbb.Dx(), sbb.Dy())
	im.Bounds = image.Rectangle{Max: sbb.Size()}
	im.Data = imageData
	im.Bitmap, _ = createImageBitmap(imageData, options)
}

// createImageBitmap calls the JS createImageBitmap function with given args.
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
	} else {
		return imageBitmap, errors.Log(errors.New("imagex.createImageBitmap failed"))
	}
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
	case gradient.Gradient:
		errors.Log(errors.New("imagex.JSImageData: setting image to Gradient not supported"))
	default:
		im.SetRGBA(AsRGBA(ui))
	}
}
