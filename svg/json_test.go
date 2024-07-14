// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"bytes"
	"testing"

	"cogentcore.org/core/base/iox/jsonx"
	"github.com/stretchr/testify/assert"
)

func TestJSON(t *testing.T) {
	r := NewRoot()
	b := bytes.Buffer{}
	assert.NoError(t, jsonx.Write(r, &b))
	assert.NoError(t, jsonx.Read(r, &b))
}
