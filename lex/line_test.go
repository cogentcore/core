// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"fmt"
	"testing"

	"goki.dev/glop/nptime"
	"goki.dev/pi/v2/token"
)

func TestLineMerge(t *testing.T) {
	l1 := Line{
		Lex{token.KeyToken{Tok: token.TextSpellErr}, 0, 10, nptime.TimeZero},
		Lex{token.KeyToken{Tok: token.TextSpellErr}, 15, 20, nptime.TimeZero},
	}
	l2 := Line{
		Lex{token.KeyToken{Tok: token.Text}, 0, 40, nptime.TimeZero},
	}

	ml := MergeLines(l1, l2)
	fmt.Printf("l1:\n%v\nl2:\n%v\nml:\n%v\n", l1, l2, ml)

	mlr := MergeLines(l2, l1)
	fmt.Printf("l1:\n%v\nl2:\n%v\nml:\n%v\n", l2, l1, mlr)
}
