// Copyright 2025 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package composer

// note: Widget RenderState -> RenderSource returns the Source

// Source is a source of input to the Composer.
type Source interface {
	Draw(c Composer)
}
