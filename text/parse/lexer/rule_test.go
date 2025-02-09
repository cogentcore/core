// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuleLexStart(t *testing.T) {
	rule := &Rule{
		Match:  String,
		String: "Hello",
		Acts:   []Actions{Next},
	}
	state := &State{
		Src: []rune("Hello, World!"),
	}

	rule.LexStart(state)

	assert.Equal(t, 0, state.Pos)
}

func TestRuleLex(t *testing.T) {
	rule := &Rule{
		Match:  String,
		String: "Hello",
		Acts:   []Actions{Next},
	}
	state := &State{
		Src: []rune("Hello, World!"),
	}

	rule.Lex(state)

	assert.Equal(t, 0, state.Pos)
}
