// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"
)

func TestSpinnerEnforceStep(t *testing.T) {
	b := NewBody()
	NewSpinner(b).SetStep(10).SetEnforceStep(true).SetValue(43)
	b.AssertRender(t, filepath.Join("spinner", "enforce-step"))
}
