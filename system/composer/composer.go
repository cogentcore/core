// Copyright 2025 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package composer

// Composer performs compositing of an ordered stack of [Source]
// elements into a target rendering destination (window).
type Composer interface {
	Start() // at start
	AddSource(sr *Source)
	Compose()
}

type ComposerBase struct {
	Sources []Source
}

// different platforms define ComposerGPU, ComposerWeb, ComposerOffscreen,
////
