// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package composer provides composition of source rendering elements.
package composer

import (
	"reflect"

	"cogentcore.org/core/base/reflectx"
)

// Composer performs composition of an ordered list of [Source]
// elements into a target rendering destination such as a window.
// Composer has two main implementations: [ComposerDrawer] and
// [ComposerWeb].
type Composer interface {

	// Start must be called at the start of each new composition run,
	// before any [Composer.Add] calls. It resets the list of [Source]
	// elements.
	Start()

	// Add adds a [Source] for the current compose run. It must be called
	// after [Composer.Start].
	Add(s Source)

	// Compose does the composition of the sources added through [Composer.Add].
	// It must be called after [Composer.Start].
	Compose()
}

// ComposerBase is the base implementation of Composer
// which manages sources.
type ComposerBase struct {
	Sources []Source
}

// different platforms define ComposerGPU, ComposerWeb, ComposerOffscreen,

func (cp *ComposerBase) Start() {
	cp.Sources = cp.Sources[:0]
}

func (cp *ComposerBase) Add(s Source) {
	if reflectx.IsNil(reflect.ValueOf(s)) {
		return
	}
	cp.Sources = append(cp.Sources, s)
}
