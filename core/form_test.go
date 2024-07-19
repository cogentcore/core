// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"github.com/stretchr/testify/assert"
)

type person struct {
	Name string
	Age  int
}

type morePerson struct {
	Name        string
	Age         int
	Job         string
	LikesGo     bool
	LikesPython bool
}

func TestForm(t *testing.T) {
	b := NewBody()
	NewForm(b).SetStruct(&person{Name: "Go", Age: 35})
	b.AssertRender(t, "form/basic")
}

func TestFormInline(t *testing.T) {
	b := NewBody()
	NewForm(b).SetInline(true).SetStruct(&person{Name: "Go", Age: 35})
	b.AssertRender(t, "form/inline")
}

func TestFormReadOnly(t *testing.T) {
	b := NewBody()
	NewForm(b).SetStruct(&person{Name: "Go", Age: 35}).SetReadOnly(true)
	b.AssertRender(t, "form/read-only")
}

func TestFormChange(t *testing.T) {
	b := NewBody()
	p := person{Name: "Go", Age: 35}

	n := 0
	value := person{}
	fm := NewForm(b).SetStruct(&p)
	fm.OnChange(func(e events.Event) {
		n++
		value = p
	})
	b.AssertRender(t, "form/change", func() {
		// [3] is value of second row, which is Age
		fm.Child(3).(*Spinner).leadingIconButton.Send(events.Click)
		assert.Equal(t, 1, n)
		assert.Equal(t, p, value)
		assert.Equal(t, person{Name: "Go", Age: 34}, p)
	})
}

func TestFormStyle(t *testing.T) {
	b := NewBody()
	s := styles.NewStyle()
	s.SetState(true, states.Active)
	s.SetAbilities(true, abilities.Checkable)
	NewForm(b).SetStruct(s)
	b.AssertRender(t, "form/style")
}
