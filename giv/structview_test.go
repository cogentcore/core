// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"testing"

	"cogentcore.org/core/gi"
)

func TestStructView(t *testing.T) {
	type myStruct struct {
		Name    string `default:"Gopher"`
		Age     int    `default:"40"`
		Rating  float32
		LikesGo bool `default:"true"`
	}
	b := gi.NewBody()
	NewStructView(b).SetStruct(&myStruct{Name: "Gopher", Age: 30, Rating: 7.3})
	b.AssertRender(t, "structview/basic")
}
