// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"fmt"
	"testing"

	"cogentcore.org/core/base/nptime"
	"cogentcore.org/core/parse/token"
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
	fmt.Printf("l1:\n%v\nl2:\n%v\nml:\n%v\n", l1, l2, ml)

	mlr := MergeLines(l2, l1)
	fmt.Printf("l1:\n%v\nl2:\n%v\nml:\n%v\n", l2, l1, mlr)
}
