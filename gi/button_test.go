// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"

	"goki.dev/girl/states"
)

func TestButton(t *testing.T) {
	for _, typ := range ButtonTypesValues() {
		for _, str := range testStrings {
			for _, ic := range testIcons {
				for _, st := range testStates {
					st := st
					var stf states.States
					stf.SetFlag(true, st...)
					nm := testName("button", typ, str, ic, stf)
					t.Run(nm, func(t *testing.T) {
						t.Parallel()
						sc := NewEmptyScene()
						NewButton(sc).SetType(typ).SetText(str).SetIcon(ic).SetState(true, st...)
						sc.AssertPixelsOnShow(t, nm)
					})
				}
			}
		}
	}
}
