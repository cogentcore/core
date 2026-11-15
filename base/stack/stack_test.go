// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stack

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack(t *testing.T) {
	var st Stack[string]
	assert.Equal(t, "", st.Pop(), "an empty stack pops the zero value")
	assert.Equal(t, "", st.Peek())
	st.Push("a")
	st.Push("b", "c")
	assert.Len(t, st, 3)
	assert.Equal(t, "c", st.Peek())
	assert.Equal(t, "c", st.Pop())
	assert.Equal(t, "b", st.Pop())
	assert.Equal(t, "a", st.Peek())
	assert.Equal(t, "a", st.Pop())
	assert.Empty(t, st)
	var ps Stack[*int]
	assert.Nil(t, ps.Pop())
}
