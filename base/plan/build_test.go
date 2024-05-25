// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type nameObj struct {
	name string
}

func (n *nameObj) Name() string {
	return n.name
}

func (n *nameObj) SetName(name string) {
	n.name = name
}

func AssertNames(t *testing.T, names []string, items []*nameObj) {
	if len(names) != len(items) {
		t.Error("lengths of lists are not the same:", len(names), len(items))
	}
	for i, nm := range names {
		inm := items[i].Name()
		if nm != inm {
			t.Error("item at index:", i, "name mismatch, should be:", nm, "was:", inm)
		}
	}
}

func TestBuild(t *testing.T) {
	var s []*nameObj

	names1 := []string{"a", "b", "c"}
	// fmt.Println("\n#### target", names1)

	r1, mods := Build(s, len(names1),
		func(i int) string { return names1[i] },
		func(name string, i int) *nameObj { return &nameObj{name: name} }, nil)

	// fmt.Println(mods, r1)
	AssertNames(t, names1, r1)
	assert.Equal(t, true, mods)

	names2 := []string{"a", "aa", "b", "c"}
	// fmt.Println("\n#### target", names2)
	r2, mods := Build(r1, len(names2),
		func(i int) string { return names2[i] },
		func(name string, i int) *nameObj {
			return &nameObj{name: name}
		}, nil)
	// fmt.Println(mods, r2)
	AssertNames(t, names2, r2)
	assert.Equal(t, true, mods)

	names3 := []string{"a", "aa", "bb", "c"}
	// fmt.Println("\n#### target", names3)
	r3, mods := Build(r2, len(names3),
		func(i int) string { return names3[i] },
		func(name string, i int) *nameObj {
			return &nameObj{name: name}
		}, nil)
	// fmt.Println(mods, r3)
	AssertNames(t, names3, r3)
	assert.Equal(t, true, mods)

	names4 := []string{"aa", "bb", "c"}
	// fmt.Println("\n#### target", names4)
	r4, mods := Build(r3, len(names4),
		func(i int) string { return names4[i] },
		func(name string, i int) *nameObj {
			return &nameObj{name: name}
		}, nil)
	// fmt.Println(mods, r4)
	AssertNames(t, names4, r4)
	assert.Equal(t, true, mods)

	r5, mods := Build(r4, len(names4),
		func(i int) string { return names4[i] },
		func(name string, i int) *nameObj {
			return &nameObj{name: name}
		}, nil)
	// fmt.Println(mods, r5)
	AssertNames(t, names4, r5)
	assert.Equal(t, false, mods)
}
