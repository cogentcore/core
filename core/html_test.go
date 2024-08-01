// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToHTML(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("Hello!")
	h, err := ToHTML(b)
	assert.NoError(t, err)
	fmt.Println(string(h))
}
