// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "testing"

func TestBind(t *testing.T) {
	b := NewBody()
	Bind("Gopher", NewTextField(b))
	b.AssertRender(t, "bind/basic")
}

func TestBindUpdate(t *testing.T) {
	b := NewBody()
	name := "Gopher"
	tf := NewTextField(b)
	Bind(&name, tf)
	b.AssertRender(t, "bind/update", func() {
		name = "Cogent Core"
		tf.Update()
	})
}
