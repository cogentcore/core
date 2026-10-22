// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import "github.com/alecthomas/chroma/v3/lexers"

func init() {
	// the tests highlight languages other than Go
	SetLexers(lexers.GlobalLexerRegistry)
}
