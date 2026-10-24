// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnAddParentWithoutScene(t *testing.T) {
	// Adding a widget to a parent that has no scene, as generating HTML
	// does for a body, keeps the scenes of the widget's descendants.
	b := NewBody()
	fr := NewFrame(b)
	tx := NewText(fr)
	sc := tx.Scene
	assert.NotNil(t, sc)

	parent := NewFrame()
	parent.AddChild(fr)
	assert.Equal(t, sc, tx.Scene)
}
