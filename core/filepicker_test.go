// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
)

func TestFilePicker(t *testing.T) {
	b := NewBody()
	// The files of the directory do not change, unlike those of the
	// current directory, which are shown by default.
	NewFilePicker(b).SetFilename("testdata/file-picker/files/")
	b.AssertRender(t, "file-picker/basic")
}
