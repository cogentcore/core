// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"errors"
	"testing"

	"github.com/alecthomas/chroma/v3"
	"github.com/stretchr/testify/assert"
)

func TestCollectTokensRecovers(t *testing.T) {
	// A lexer that fails panics from its iterator, as chroma does on a
	// regular expression match timeout.
	failure := errors.New("match timeout")
	failing := func(yield func(chroma.Token) bool) {
		if !yield(chroma.Token{Type: chroma.Keyword, Value: "func"}) {
			return
		}
		panic(failure)
	}
	toks, err := collectTokens(failing)
	assert.Nil(t, toks)
	assert.ErrorIs(t, err, failure)

	// A lexer that succeeds gives all of its tokens.
	ok := func(yield func(chroma.Token) bool) {
		yield(chroma.Token{Type: chroma.Keyword, Value: "func"})
	}
	toks, err = collectTokens(ok)
	assert.NoError(t, err)
	assert.Len(t, toks, 1)
}
