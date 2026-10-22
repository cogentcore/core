// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package highlighting

import "github.com/alecthomas/chroma/v3/lexers"

func init() {
	// all of the chroma lexers by default on native platforms
	lexerRegistry = lexers.GlobalLexerRegistry
}
