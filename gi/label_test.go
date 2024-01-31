// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"
)

func TestLabel(t *testing.T) {
	for _, typ := range LabelTypesValues() {
		for _, str := range testStrings {
			if str == "" {
				continue
			}
			b := NewBody()
			NewLabel(b).SetType(typ).SetText(str)
			b.AssertRender(t, testName("label", typ, str))
		}
	}
}
