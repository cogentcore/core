// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateTest(t *testing.T) {
	t.Skip("only when regenerating")
	items, err := Open("/Users/oreilly/ccnlab_bib/ccnlab.json")
	fullTest := false
	assert.NoError(t, err)
	types := make(map[int]bool)
	var nit []Item
	for i := range items {
		it := &items[i]
		hash := int(it.Type) * 100000
		if fullTest {
			hash += max(len(it.Editor), 3)*1000 + max(len(it.Author), 3)*100
			if it.Edition != "" {
				hash += 1
			}
			if it.URL != "" || it.DOI != "" {
				hash += 2
			}
			if it.Page != "" || it.Volume != "" {
				hash += 4
			}
		}
		_, ok := types[hash]
		if ok {
			continue
		}
		types[hash] = true
		nit = append(nit, *it)
	}
	err = SaveItems(nit, "testdata/test.json")
	assert.NoError(t, err)
}

func TestOpen(t *testing.T) {
	items, err := Open("testdata/test.json")
	assert.NoError(t, err)
	err = SaveItems(items, "testdata/save.json")
	assert.NoError(t, err)
	titems, err := Open("testdata/save.json")
	assert.NoError(t, err)
	assert.Equal(t, items, titems)
	// items, err = Open("/Users/oreilly/ccnlab_bib/ccnlab.json")
	// assert.NoError(t, err)
}

func TestCiteAPA(t *testing.T) {
	t.Skip("just for development")
	items, err := Open("testdata/test.json")
	assert.NoError(t, err)
	for i := range items {
		it := &items[i]
		// if it.Type <= Book || it.Type == Chapter || it.Type == PaperConference || it.Type == Thesis {
		// 	continue
		// }
		// if it.Type != PaperConference {
		// 	continue
		// }
		cite := CiteAPA(Parenthetical, it)
		ref := RefAPA(it)
		fmt.Println(it.Type.String()+": ", cite)
		fmt.Println(ref)
		fmt.Println("")
	}
}
