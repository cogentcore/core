// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fonts

import "io/fs"

// EmbeddedFonts are embedded filesystems to get fonts from. By default,
// this includes a set of Roboto and Roboto Mono fonts. System fonts are
// automatically supported. This is not relevant on web, which uses available
// web fonts. Use [AddEmbeddedFonts] to add to this. This must be called before
// [NewShaper] to have an effect.
var EmbeddedFonts = []fs.FS{DefaultFonts}

// AddEmbeddedFonts adds to [EmbeddedFonts] for font loading.
func AddEmbeddedFonts(fsys ...fs.FS) {
	EmbeddedFonts = append(EmbeddedFonts, fsys...)
}
