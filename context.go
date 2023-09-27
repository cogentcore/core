// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import "image/color"

// Context contains information about the context in which color parsing occurs.
type Context interface {
	// Base returns the base color that the color parsing is relative top
	Base() color.Color
	// FullByURL returns the [Full] color with the given URL.
	// A URL of "#name" is typical, where name
	// is the name of a node with a [Full] color in it.
	FullByURL(url string) *Full
}
