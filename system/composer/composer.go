// Copyright 2025 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package composer

import (
	"reflect"

	"cogentcore.org/core/base/reflectx"
)

// Composer performs compositing of an ordered stack of [Source]
// elements into a target rendering destination (e.g., window).
type Composer interface {

	// Start is called at the start of a new compose run
	Start()

	// Add adds a source for the current compose run
	Add(s Source)

	// Compose does the composition of the added sources.
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
