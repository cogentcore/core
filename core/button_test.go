// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
)

func TestButton(t *testing.T) {
	for typi, typ := range ButtonTypesValues() {
		for stri, str := range testStrings {
			for ici, ic := range testIcons {
				if stri == 0 && ici == 0 {
					continue
				}
				for sti, st := range testStates {
					// we do not test other types and states of the rest
					// of the strings, as that is a waste of
					// testing time
					if stri > 1 && (typi > 0 || sti > 0) {
						continue
					}
					b := NewBody()
					bt := NewButton(b).SetType(typ).SetText(str).SetIcon(ic).SetState(true, st...)
					nm := testName("button", typ, str, ic, bt.Styles.State)
					b.AssertRender(t, nm)
				}
			}
		}
	}
}
