// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

type File struct {
	Lines [][]rune  `desc:"contents of the file as lines of runes"`
	Lexs  []LexLine `desc:"lex'd version of the lines"`
}
