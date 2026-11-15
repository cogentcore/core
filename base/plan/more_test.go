// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plan

//core:js-test

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// update updates the items to the given names, recording the new,
// initialized, and destroyed items.
func update(s *[]*nameObj, names []string, made, inited, destroyed *[]string) bool {
	return Update(s, len(names),
		func(i int) string { return names[i] },
		func(name string, i int) *nameObj {
			*made = append(*made, name)
			return &nameObj{name: name}
		},
		func(e *nameObj, i int) {
			*inited = append(*inited, e.name+"@"+strconv.Itoa(i))
		},
		func(e *nameObj) {
			*destroyed = append(*destroyed, e.name)
		})
}

func TestUpdateFuncs(t *testing.T) {
	var s []*nameObj
	var made, inited, destroyed []string
	assert.True(t, update(&s, []string{"a", "b", "c"}, &made, &inited, &destroyed))
	assert.Equal(t, []string{"a", "b", "c"}, made)
	assert.Equal(t, []string{"a@0", "b@1", "c@2"}, inited)
	a, c := s[0], s[2]

	made, inited = nil, nil
	assert.True(t, update(&s, []string{"c", "d", "a"}, &made, &inited, &destroyed))
	assertNames(t, []string{"c", "d", "a"}, s)
	assert.Equal(t, []string{"d"}, made)
	assert.Equal(t, []string{"d@1"}, inited)
	assert.Equal(t, []string{"b"}, destroyed)
	assert.Same(t, c, s[0], "existing items are moved, not made again")
	assert.Same(t, a, s[2])

	assert.False(t, update(&s, []string{"c", "d", "a"}, &made, &inited, &destroyed))
	assert.True(t, update(&s, nil, &made, &inited, &destroyed))
	assert.Empty(t, s)
	assert.Panics(t, func() { update(&s, []string{"x", "x"}, &made, &inited, &destroyed) })
}

// TestUpdateRandom tests updating to random plans, which always gives
// the planned names, keeping the existing items.
func TestUpdateRandom(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	var s []*nameObj
	var made, inited, destroyed []string
	for range 200 {
		perm := rnd.Perm(10)[:rnd.Intn(10)]
		names := make([]string, len(perm))
		for i, p := range perm {
			names[i] = strconv.Itoa(p)
		}
		old := map[string]*nameObj{}
		for _, e := range s {
			old[e.name] = e
		}
		update(&s, names, &made, &inited, &destroyed)
		assertNames(t, names, s)
		for _, e := range s {
			if o, ok := old[e.name]; ok {
				assert.Same(t, o, e)
			}
		}
	}
}
