// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	var s []*nameObj

	config1 := Config[*nameObj]{
		{"a", func() *nameObj { return &nameObj{} }},
		{"b", func() *nameObj { return &nameObj{} }},
		{"c", func() *nameObj { return &nameObj{} }},
	}
	// fmt.Println("\n#### target", config1)
	r1, mods := UpdateConfig(s, config1)
	// fmt.Println(mods, r1)

	names1 := []string{"a", "b", "c"}
	AssertNames(t, names1, r1)
	assert.Equal(t, true, mods)
}