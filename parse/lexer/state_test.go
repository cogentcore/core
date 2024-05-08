// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadUntil(t *testing.T) {
	ls := &State{
		Src: []rune(" ( Hello } , ) ] Worabcld!"),
		Pos: 0,
		Ch:  'H',
	}

	ls.ReadUntil("(")
	assert.Equal(t, 2, ls.Pos)

	ls.ReadUntil("}")
	assert.Equal(t, 10, ls.Pos)

	ls.ReadUntil(")")
	assert.Equal(t, 14, ls.Pos)

	ls.ReadUntil("]")
	assert.Equal(t, 16, ls.Pos)

	ls.ReadUntil("abc")
	assert.Equal(t, 23, ls.Pos)

	ls.ReadUntil("h")
	assert.Equal(t, 26, ls.Pos)
}
