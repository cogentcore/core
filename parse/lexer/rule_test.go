// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"testing"

	"cogentcore.org/core/parse/token"
	"github.com/stretchr/testify/assert"
)

func TestRuleLexStart(t *testing.T) {
	rule := &Rule{}
	state := &State{Lex: Line{
		{Token: token.KeyToken{Token: token.Text}},
		{Token: token.KeyToken{Token: token.Text}},
	}}

	result := rule.LexStart(state)

	assert.Equal(t, &Rule{}, result)
	assert.Equal(t, 0, state.Pos)
	assert.Equal(t, 2, len(state.Lex))
}
