// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package alllexers makes syntax highlighting use all of the chroma lexers
// on all platforms, when imported:
//
//	import _ "cogentcore.org/core/text/highlighting/alllexers"
//
// This is already the default on native platforms; on the web, the default
// is [highlighting.NewCoreLexers], which leaves out the definitions of the
// hundreds of other languages that chroma supports, saving several
// megabytes in the size of the app.
package alllexers

import (
	"cogentcore.org/core/text/highlighting"
	"github.com/alecthomas/chroma/v3/lexers"
)

func init() {
	highlighting.SetLexers(lexers.GlobalLexerRegistry)
}
