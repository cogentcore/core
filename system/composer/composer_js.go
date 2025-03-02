// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package composer

import (
	"reflect"
	"syscall/js"
)

// ComposerWeb is the web implementation of [Composer].
type ComposerWeb struct {

	// Sources are the composition [Source]s.
	Sources []Source

	// Pointers are the pointers for source context values, with indices
	// in one-to-one correspondence with [ComposerWeb.Sources].
	// These pointers are not actually used for dereferencing anything; they
	// are merely a unique identifier. This is necessary because the [Source]
	// is new each time, but the context pointer stays the same.
	Pointers map[Source]uint64

	// Elements are the HTML elements coresponding to each pointer for source context values
	// (see [ComposerWeb.Pointers]). Unlike [ComposerWeb.Sources]
	// and [ComposerWeb.Pointers], this is persistent to [ComposerWeb.Start], as this allows
	// us to re-use the same HTML elements when possible.
	Elements map[uint64]js.Value
}

func (cw *ComposerWeb) Start() {
	cw.Sources = cw.Sources[:0]
	clear(cw.Pointers)
}

func (cw *ComposerWeb) Add(s Source, ctx any) {
	if s == nil {
		return
	}
	cw.Sources = append(cw.Sources, s)
	cw.Pointers[s] = uint64(reflect.ValueOf(ctx).Pointer())
}

func (cw *ComposerWeb) Compose() {
	for _, s := range cw.Sources {
		s.Draw(cw)
	}
}

// Element returns the HTML element for the given [Source], making it with the given
// tag if it doesn't exist yet.
func (cw *ComposerWeb) Element(s Source, tag string) js.Value {
	ptr := cw.Pointers[s]
	elem := cw.Elements[ptr]
	if !elem.IsUndefined() {
		return elem
	}
	// TODO: offscreen canvas?
	document := js.Global().Get("document")
	elem = document.Call("createElement", tag)
	document.Get("body").Call("appendChild", elem)
	cw.Elements[ptr] = elem
	return elem
}
