// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package alllexers makes syntax highlighting use all of the chroma lexers
// on all platforms, when imported:
//
//	import _ "cogentcore.org/core/text/highlighting/alllexers"
//
// An editor of code in any language needs this. Otherwise, the default is
// [highlighting.NewGoLexers], which highlights only Go and leaves out the
// definitions of the hundreds of other languages that chroma supports,
// saving several megabytes in the size of the app, and
// [highlighting.AddLexers] adds the lexers of particular languages.
package alllexers

import (
	"cogentcore.org/core/text/highlighting"
	"github.com/alecthomas/chroma/v3/lexers"
)

func init() {
	highlighting.SetLexers(lexers.GlobalLexerRegistry)
}
