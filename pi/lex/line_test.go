// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"fmt"
	"testing"

	"cogentcore.org/core/gox/nptime"
	"cogentcore.org/core/pi/token"
)

func TestLineMerge(t *testing.T) {
	l1 := Line{
		Lex{token.KeyToken{Token: token.TextSpellErr}, 0, 10, nptime.TimeZero},
		Lex{token.KeyToken{Token: token.TextSpellErr}, 15, 20, nptime.TimeZero},
	}
	l2 := Line{
		Lex{token.KeyToken{Token: token.Text}, 0, 40, nptime.TimeZero},
	}

	ml := MergeLines(l1, l2)
	fmt.Printf("l1:\n%v\nl2:\n%v\nml:\n%v\n", l1, l2, ml)

	mlr := MergeLines(l2, l1)
	fmt.Printf("l1:\n%v\nl2:\n%v\nml:\n%v\n", l2, l1, mlr)
}
