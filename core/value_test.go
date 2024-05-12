// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/stretchr/testify/assert"
)

func TestBind(t *testing.T) {
	b := NewBody()
	Bind("Gopher", NewTextField(b))
	b.AssertRender(t, "bind/basic")
}

func TestBindUpdate(t *testing.T) {
	b := NewBody()
	name := "Gopher"
	tf := Bind(&name, NewTextField(b))
	b.AssertRender(t, "bind/update", func() {
		name = "Cogent Core"
		tf.Update()
	})
}

func TestBindChange(t *testing.T) {
	b := NewBody()
	name := "Gopher"

	tf := Bind(&name, NewTextField(b))
	b.AssertRender(t, "bind/change", func() {
		tf.HandleEvent(events.NewKey(events.KeyChord, 'G', 0, 0))
		tf.HandleEvent(events.NewKey(events.KeyChord, 'o', 0, 0))
		tf.HandleEvent(events.NewKey(events.KeyChord, ' ', 0, 0))
		assert.Equal(t, "Gopher", name)
		tf.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
		assert.Equal(t, "Go Gopher", name)
	})
}
