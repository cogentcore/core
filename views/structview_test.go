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

func TestStructView(t *testing.T) {
	b := core.NewBody()
	NewStructView(b).SetStruct(&person{Name: "Go", Age: 35})
	b.AssertRender(t, "struct-view/basic")
}

func TestStructViewInline(t *testing.T) {
	b := core.NewBody()
	NewStructView(b).SetInline(true).SetStruct(&person{Name: "Go", Age: 35})
	b.AssertRender(t, "struct-view/inline")
}

func TestStructViewReadOnly(t *testing.T) {
	b := core.NewBody()
	NewStructView(b).SetStruct(&person{Name: "Go", Age: 35}).SetReadOnly(true)
	b.AssertRender(t, "struct-view/read-only")
}

func TestStructViewChange(t *testing.T) {
	b := core.NewBody()
	p := person{Name: "Go", Age: 35}

	n := 0
	value := person{}
	sv := NewStructView(b).SetStruct(&p)
	sv.OnChange(func(e events.Event) {
		n++
		value = p
	})
	b.AssertRender(t, "struct-view/change", func() {
		// [3] is value of second row, which is Age
		sv.Child(3).(*core.Spinner).LeadingIconButton().Send(events.Click)
		assert.Equal(t, 1, n)
		assert.Equal(t, p, value)
		assert.Equal(t, person{Name: "Go", Age: 34}, p)
	})
}
