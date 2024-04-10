// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"testing"

	"cogentcore.org/core/core"
)

type myStruct struct {
	Name    string `default:"Gopher"`
	Age     int    `default:"40"`
	Rating  float32
	LikesGo bool `default:"true"`
}

var myStructValue = &myStruct{Name: "Gopher", Age: 30, Rating: 7.3}

func TestStructView(t *testing.T) {
	b := core.NewBody()
	NewStructView(b).SetStruct(myStructValue)
	b.AssertRender(t, "structview/basic")
}
