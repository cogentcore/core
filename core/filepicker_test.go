// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
)

func TestFilePicker(t *testing.T) {
	t.Skip("todo: randomly not working, https://github.com/cogentcore/core/issues/1641")
	b := NewBody()
	NewFilePicker(b)
	b.AssertRender(t, "file-picker/basic")
}
