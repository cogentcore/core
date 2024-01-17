// Copyright (c) 2023, Cogent Core. All rights reserved.
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
	// ImageByURL returns the [image.Image] associated with the given URL.
	// Typical URL formats are HTTP URLs like "https://example.com" and node
	// URLs like "#name". If it returns nil, that indicats that there is no
	// [image.Image] color associated with the given URL.
	ImageByURL(url string) image.Image
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

func (bc *baseContext) ImageByURL(url string) image.Image {
	return nil
}
