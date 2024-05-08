// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"testing"

	"cogentcore.org/core/base/nptime"
	"cogentcore.org/core/parse/token"
	"github.com/stretchr/testify/assert"
)

func TestLineMerge(t *testing.T) {
	l1 := Line{
		Lex{token.KeyToken{Token: token.TextSpellErr}, 0, 10, nptime.Time{}},
		Lex{token.KeyToken{Token: token.TextSpellErr}, 15, 20, nptime.Time{}},
	}
	l2 := Line{
		Lex{token.KeyToken{Token: token.Text}, 0, 40, nptime.Time{}},
	}

	ml := MergeLines(l1, l2)
	assert.Equal(t, Line{
		Lex{token.KeyToken{Token: token.Text}, 0, 40, nptime.Time{}},
		Lex{token.KeyToken{Token: token.TextSpellErr}, 0, 10, nptime.Time{}},
		Lex{token.KeyToken{Token: token.TextSpellErr}, 15, 20, nptime.Time{}},
	}, ml)

	mlr := MergeLines(l2, l1)
	assert.Equal(t, Line{
		Lex{token.KeyToken{Token: token.Text}, 0, 40, nptime.Time{}},
		Lex{token.KeyToken{Token: token.TextSpellErr}, 0, 10, nptime.Time{}},
		Lex{token.KeyToken{Token: token.TextSpellErr}, 15, 20, nptime.Time{}},
	}, mlr)
}
