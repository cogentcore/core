// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package imagex

import (
	"bytes"
	"image"
	"io"
	"syscall/js"

	"cogentcore.org/core/base/jscp"
)

// browserFormats are the image formats that the browser decodes itself,
// which is much faster than decoding them in Go on the web.
var browserFormats = map[string]bool{"png": true, "jpeg": true, "gif": true, "webp": true, "bmp": true}

// decode decodes an image from the given reader. On the web, formats that
// the browser supports are decoded by the browser with createImageBitmap,
// into a [JSRGBA] whose pixels are only read back when they are needed,
// and other formats, or images the browser fails to decode, are decoded
// in Go.
func decode(r io.Reader) (image.Image, string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, "", err
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err == nil && browserFormats[format] {
		if im := decodeBrowser(b, cfg); im != nil {
			return im, format, nil
		}
	}
	return image.Decode(bytes.NewReader(b))
}

// decodeBrowser decodes the given encoded image bytes with the browser's
// createImageBitmap, returning nil if that fails.
func decodeBrowser(b []byte, cfg image.Config) *JSRGBA {
	// Runtimes without createImageBitmap, such as bun, decode in Go.
	if js.Global().Get("createImageBitmap").IsUndefined() {
		return nil
	}
	blob := js.Global().Get("Blob").New([]any{jscp.BytesToJS(b)})
	bitmap, err := createImageBitmap(blob, nil)
	if err != nil {
		return nil
	}
	im := &JSRGBA{}
	im.JS.Bitmap = bitmap
	im.JS.Bounds = image.Rect(0, 0, cfg.Width, cfg.Height)
	return im
}

// readBitmapPixels reads the pixels of the given ImageBitmap back by
// drawing it on an offscreen canvas.
func readBitmapPixels(bitmap js.Value, bounds image.Rectangle) *image.RGBA {
	w, h := bounds.Dx(), bounds.Dy()
	canvas := js.Global().Get("OffscreenCanvas").New(w, h)
	ctx := canvas.Call("getContext", "2d")
	ctx.Call("drawImage", bitmap, 0, 0)
	data := ctx.Call("getImageData", 0, 0, w, h).Get("data")
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	js.CopyBytesToGo(src.Pix, js.Global().Get("Uint8Array").New(data.Get("buffer"), data.Get("byteOffset"), data.Get("length")))
	return AsRGBA(src)
}
