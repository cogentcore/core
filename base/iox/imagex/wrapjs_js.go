// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package imagex

import (
	"image"
	"syscall/js"

	"cogentcore.org/core/base/errors"
	"github.com/cogentcore/webgpu/jsx"
)

// WrapJS returns a JavaScript optimized wrapper around the given
// image.Image on web platform, and just returns the image otherwise.
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
		return NewJSRGBA(x)
	case *image.NRGBA:
		return NewJSRGBA(x)
	default:
		return src
	}
}

// JSRGBA is a wrapper around image.RGBA that adds JSImageData pointers.
// It implements the imagex.Image wrapping interface.
type JSRGBA struct {
	image.RGBA
	JS JSImageData
}

// NewJSRGBA returns a new wrapped JSRGBA image from original source image.
// If the source is already a wrapped JSRGBA image, it returns a shallow
// copy of that original data, re-using the pixels and js pointers.
// Returns nil if the source image is nil.
func NewJSRGBA(src image.Image) *JSRGBA {
	if src == nil {
		return nil
	}
	im := &JSRGBA{}
	if x, ok := src.(*JSRGBA); ok {
		*im = *x
		return im
	}
	im.RGBA = *AsRGBA(src)
	im.Update()
	return im
}

// Update must be called any time the image has been updated!
func (im *JSRGBA) Update() {
	im.JS.SetRGBA(&im.RGBA)
}

func (im *JSRGBA) Underlying() image.Image {
	return &im.RGBA
}

//////// JSImageData

// JSImageData has JavaScript pointers to the image bytes and an ImageBitmap
// (gpu texture) of an image.
type JSImageData struct {
	// Data is the raw Pix bytes in a javascript array buffer.
	Data js.Value

	// Bitmap is the result of createImageBitmap on ImageData; a gpu texture basically.
	Bitmap js.Value
}

// setImageData sets the JavaScript pointers from given bytes.
func (im *JSImageData) SetImageData(src []byte, sbb image.Rectangle, options map[string]any) {
	jsBuf := jsx.BytesToJS(src)
	imageData := js.Global().Get("ImageData").New(jsBuf, sbb.Dx(), sbb.Dy())
	im.Data = imageData
	imageBitmapPromise := js.Global().Call("createImageBitmap", imageData, options)
	imageBitmap, ok := jsx.Await(imageBitmapPromise)
	if ok {
		im.Bitmap = imageBitmap
	} else {
		errors.Log(errors.New("imagex.JSImageData: createImageBitmap failed"))
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
	}
}
