// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package pimage

import (
	"image"
	"syscall/js"

	"github.com/cogentcore/webgpu/wgpu"
)

func (pr *Params) UpdateSource() {
	setJSImageBitmap(pr)
}

func setJSImageBitmap(pr *Params) {
	// TODO: support [*image.NRGBA]
	if src, ok := pr.Source.(*image.RGBA); ok {
		jsBuf := wgpu.BytesToJS(src.Pix)
		sbb := pr.Source.Bounds()
		imageData := js.Global().Get("ImageData").New(jsBuf, sbb.Dx(), sbb.Dy())
		pr.jsImageData = imageData
		imageBitmapPromise := js.Global().Call("createImageBitmap", imageData)
		imageBitmap, ok := jsAwait(imageBitmapPromise)
		if ok {
			pr.jsImageBitmap = imageBitmap
		}
	}
}

func GetJSImageData(pr *Params) js.Value {
	if pr.jsImageData == nil {
		return js.Undefined()
	}
	return pr.jsImageData.(js.Value)
}

func GetJSImageBitmap(pr *Params) js.Value {
	if pr.jsImageBitmap == nil {
		return js.Undefined()
	}
	return pr.jsImageBitmap.(js.Value)
}

func jsAwait(v js.Value) (result js.Value, ok bool) { // TODO: use wgpu version
	// COPIED FROM https://go-review.googlesource.com/c/go/+/150917/
	if v.Type() != js.TypeObject || v.Get("then").Type() != js.TypeFunction {
		return v, true
	}

	done := make(chan struct{})

	onResolve := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result = args[0]
		ok = true
		close(done)
		return nil
	})
	defer onResolve.Release()

	onReject := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result = args[0]
		ok = false
		close(done)
		return nil
	})
	defer onReject.Release()

	v.Call("then", onResolve, onReject)
	<-done
	return
}
