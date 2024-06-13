// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
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
	b := core.NewBody()
	NewForm(b).SetStruct(&person{Name: "Go", Age: 35})
	b.AssertRender(t, "form/basic")
}

func TestFormInline(t *testing.T) {
	b := core.NewBody()
	NewForm(b).SetInline(true).SetStruct(&person{Name: "Go", Age: 35})
	b.AssertRender(t, "form/inline")
}

func TestFormReadOnly(t *testing.T) {
	b := core.NewBody()
	NewForm(b).SetStruct(&person{Name: "Go", Age: 35}).SetReadOnly(true)
	b.AssertRender(t, "form/read-only")
}

func TestFormChange(t *testing.T) {
	b := core.NewBody()
	p := person{Name: "Go", Age: 35}

	n := 0
	value := person{}
	sv := NewForm(b).SetStruct(&p)
	sv.OnChange(func(e events.Event) {
		n++
		value = p
	})
	b.AssertRender(t, "form/change", func() {
		// [3] is value of second row, which is Age
		sv.Child(3).(*core.Spinner).LeadingIconButton().Send(events.Click)
		assert.Equal(t, 1, n)
		assert.Equal(t, p, value)
		assert.Equal(t, person{Name: "Go", Age: 34}, p)
	})
}
