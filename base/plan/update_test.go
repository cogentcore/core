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

func (n *nameObj) PlanName() string {
	return n.name
}

func assertNames(t *testing.T, names []string, items []*nameObj) {
	if len(names) != len(items) {
		t.Error("lengths of lists are not the same:", len(names), len(items))
	}
	for i, nm := range names {
		inm := items[i].PlanName()
		if nm != inm {
			t.Error("item at index:", i, "name mismatch, should be:", nm, "was:", inm)
		}
	}
}

func TestUpdate(t *testing.T) {
	var s []*nameObj

	names1 := []string{"a", "b", "c"}

	changed := Update(&s, len(names1),
		func(i int) string { return names1[i] },
		func(name string, i int) *nameObj { return &nameObj{name: name} }, nil, nil)

	assertNames(t, names1, s)
	assert.Equal(t, true, changed)

	names2 := []string{"a", "aa", "b", "c"}
	changed = Update(&s, len(names2),
		func(i int) string { return names2[i] },
		func(name string, i int) *nameObj {
			return &nameObj{name: name}
		}, nil, nil)
	assertNames(t, names2, s)
	assert.Equal(t, true, changed)

	names3 := []string{"a", "aa", "bb", "c"}
	changed = Update(&s, len(names3),
		func(i int) string { return names3[i] },
		func(name string, i int) *nameObj {
			return &nameObj{name: name}
		}, nil, nil)
	assertNames(t, names3, s)
	assert.Equal(t, true, changed)

	names4 := []string{"aa", "bb", "c"}
	changed = Update(&s, len(names4),
		func(i int) string { return names4[i] },
		func(name string, i int) *nameObj {
			return &nameObj{name: name}
		}, nil, nil)
	assertNames(t, names4, s)
	assert.Equal(t, true, changed)

	changed = Update(&s, len(names4),
		func(i int) string { return names4[i] },
		func(name string, i int) *nameObj {
			return &nameObj{name: name}
		}, nil, nil)
	assertNames(t, names4, s)
	assert.Equal(t, false, changed)
}
