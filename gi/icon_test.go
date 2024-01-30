// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"

	"cogentcore.org/core/icons"
)

func TestIconBasic(t *testing.T) {
	b := NewBody()
	NewIcon(b).SetIcon(icons.Close)
	b.AssertRender(t, filepath.Join("icon", "basic"))
}
