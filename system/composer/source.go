// Copyright 2025 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package composer

// Source is a source of input to the Composer.
// Different platforms define Draw functions to draw
// to a platform-appropriate destination. The source
// object MUST be fully self-contained and not have
// any pointers into other state: it should fully describe
// the rendering, and be runnable in a separate goroutine.
type Source interface {
	Draw(c Composer)
}
