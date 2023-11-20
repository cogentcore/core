// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "testing"

func TestButton(t *testing.T) {
	for _, typ := range ButtonTypesValues() {
		for _, str := range testStrings {
			for _, ic := range testIcons {
				for _, st := range testStates {
					sc := NewEmptyScene()
					bt := NewButton(sc).SetType(typ).SetText(str).SetIcon(ic).SetState(true, st...)
					nm := testName("button", typ, str, ic, bt.Styles.State)
					sc.AssertPixelsOnShow(t, nm)
				}
			}
		}
	}
}
