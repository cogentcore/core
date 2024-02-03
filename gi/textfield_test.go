// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "testing"

func TestTextField(t *testing.T) {
	for _, typ := range TextFieldTypesValues() {
		for _, str := range testStrings {
			for _, lic := range testIcons {
				for _, tic := range testIcons1 {
					b := NewBody()
					NewTextField(b).SetType(typ).SetText(str).SetLeadingIcon(lic).SetTrailingIcon(tic)
					nm := testName("textfield", typ, str, lic, tic)
					b.AssertRender(t, nm)
				}
			}
		}
	}
}
