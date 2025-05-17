// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package composer provides composition of source rendering elements.
package composer

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
	// after [Composer.Start]. It takes a context argument, which is used
	// to create a unique identifier in [ComposerWeb], so the context argument
	// must simply be a pointer unique to this [Source] (the pointer is not
	// actually used for dereferencing anything).
	Add(s Source, ctx any)

	// Compose does the composition of the sources added through [Composer.Add].
	// It must be called after [Composer.Start].
	Compose()

	// Redraw re-renders the last composition. This is called during
	// resizing, for example.
	Redraw()
}
