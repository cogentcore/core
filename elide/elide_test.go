// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package elide

import (
	"fmt"
	"testing"
)

func TestElide(t *testing.T) {
	s := "string for testing purposes"
	e := End(s, 7)
	fmt.Println(len(e), e)
	if len(e) > 7 {
		t.Error("len should not be more than 7", len(e))
	}
	m := Middle(s, 7)
	fmt.Println(len(m), m)
	if len(m) > 7 {
		t.Error("len should not be more than 7", len(m))
	}
}
