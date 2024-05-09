// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"testing"

	"cogentcore.org/core/core"
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

func TestStructViewReadOnly(t *testing.T) {
	b := core.NewBody()
	NewStructView(b).SetStruct(&person{Name: "Go", Age: 35}).SetReadOnly(true)
	b.AssertRender(t, "struct-view/read-only")
}
