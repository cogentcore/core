// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package elide

import (
	"testing"
)

func TestElide(t *testing.T) {
	s := "string for testing purposes"
	have := End(s, 7)
	want := ""
	if have != want {
		t.Errorf("expected %q got %q", want, have)
	}

	have = Middle(s, 7)
	want = ""
	if have != want {
		t.Errorf("expected %q got %q", want, have)
	}
}
