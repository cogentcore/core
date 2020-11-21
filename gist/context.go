// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

// Context provides external contextual information needed for styles
type Context interface {
	// ContextColor returns the current Color activated in the context.
	// Color has support for special color names that are relative to
	// this current color.
	ContextColor() Color

	// ContextColorSpecByURL returns a ColorSpec from given URL expression
	// used in color specifications: url(#name) is typical, where name
	// is the name of a node with a ColorSpec in it.
	ContextColorSpecByURL(url string) *ColorSpec
}
