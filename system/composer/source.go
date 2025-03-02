// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package composer

// Source is a source of input to a [Composer].
// Sources typically define multiple draw functions behind build tags
// or type assertions to provide platform-specific implementations for
// the two main [Composer] implementations ([ComposerDrawer] and [ComposerWeb]);
// see core/render_js.go and core/render_notjs.go for examples.
//
// The source object MUST be fully self-contained and not have
// any pointers into other state: it should fully describe
// the rendering, and be runnable in a separate goroutine.
type Source interface {

	// Draw draws the source to the given [Composer].
	Draw(c Composer)
}
