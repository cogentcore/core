// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image"
	"image/color"
)

// Context contains information about the context in which color parsing occurs.
type Context interface {
	// Base returns the base color that the color parsing is relative top
	Base() color.RGBA
	// FullByURL returns the [image.Image] color with the given URL.
	// A URL of "#name" is typical, where name is the name of a node
	// with an [image.Image] color in it. If it returns nil, that
	// indicats that there is no [image.Image] color associated with
	// the given URL.
	ColorByURL(url string) image.Image
}

// BaseContext returns a basic [Context] based on the given base color.
func BaseContext(base color.RGBA) Context {
	return &baseContext{base}
}

type baseContext struct {
	base color.RGBA
}

func (bc *baseContext) Base() color.RGBA {
	return bc.base
}

func (bc *baseContext) ColorByURL(url string) image.Image {
	return nil
}
