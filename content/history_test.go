// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHistorySaveRepeat tests that saving the current location again,
// as opening the current page does, keeps the index on the current location.
func TestHistorySaveRepeat(t *testing.T) {
	a := &Location{URL: "a"}
	b := &Location{URL: "b"}
	c := &Location{URL: "c"}
	hs := &History{}
	hs.Save(a)
	hs.Save(&Location{URL: "a"})
	assert.Equal(t, 0, hs.Index)
	assert.Len(t, hs.Records, 1)
	hs.Save(b)
	hs.Save(&Location{URL: "b"})
	assert.Equal(t, 1, hs.Index)
	assert.Len(t, hs.Records, 2)
	hs.Back()
	hs.Save(&Location{URL: "a"})
	assert.Equal(t, 0, hs.Index)
	assert.Len(t, hs.Records, 2, "a repeat keeps the forward history")
	hs.Save(c)
	assert.Equal(t, 1, hs.Index)
	assert.Equal(t, []*Location{a, c}, hs.Records)
}
