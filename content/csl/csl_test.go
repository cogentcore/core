// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpen(t *testing.T) {
	it, err := Open("testdata/test.json")
	assert.NoError(t, err)
	err = Save(it, "testdata/save.json")
	assert.NoError(t, err)
	tit, err := Open("testdata/save.json")
	assert.NoError(t, err)
	assert.Equal(t, it, tit)
	// it, err = Open("/Users/oreilly/ccnlab_bib/ccnlab.json")
	// assert.NoError(t, err)
}

func TestCiteAPA(t *testing.T) {
	items, err := Open("testdata/test.json")
	assert.NoError(t, err)
	for i := range items {
		it := &items[i]
		apa := CiteAPA(it)
		fmt.Println(apa)
		fmt.Println("")
	}
}
