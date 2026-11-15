// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/events/key"
	"cogentcore.org/core/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sendSplitsKey sends the platform's command chord for the given digit
// to the splits.
func sendSplitsKey(sp *Splits, digit int) {
	mod := key.Control
	if TheApp.Platform() == system.MacOS {
		mod = key.Meta
	}
	code := key.Code1 + key.Codes(digit-1)
	if digit == 0 {
		code = key.Code0
	}
	SendKey(sp, rune('0'+digit), code, mod)
}

// assertSplits asserts the splits of the children.
func assertSplits(t *testing.T, sp *Splits, want ...float32) {
	t.Helper()
	got := sp.Splits()
	require.Len(t, got, len(want))
	for i := range want {
		assert.InDelta(t, want[i], got[i], 0.001, "split %d: %v", i, got)
	}
}

// TestSplitsCollapseKeys tests collapsing and restoring tiles with the
// number keys, and saving and restoring splits.
func TestSplitsCollapseKeys(t *testing.T) {
	b, sp := makeSplits(3, 40, 20)
	AssertRender(t, b, "splits/collapse-keys", Locked(b, func() {
		assertSplits(t, sp, 1.0/3, 1.0/3, 1.0/3)
		sendSplitsKey(sp, 2)
		assertSplits(t, sp, 0.5, 0, 0.5)
		assert.Equal(t, []float32{1.0 / 3, 1.0 / 3, 1.0 / 3}, sp.savedSplits)

		sendSplitsKey(sp, 2)                    // restores it
		assertSplits(t, sp, 0.375, 0.25, 0.375) // a third, normalized
		sendSplitsKey(sp, 0)
		assertSplits(t, sp, 1.0/3, 1.0/3, 1.0/3)
		sendSplitsKey(sp, 9) // no tile
		assertSplits(t, sp, 1.0/3, 1.0/3, 1.0/3)

		sp.collapseSplit(true, 0)
		assertSplits(t, sp, 0, 0.5, 0.5)
		sp.restoreSplits()
		assertSplits(t, sp, 1.0/3, 1.0/3, 1.0/3)
		sp.collapseSplit(false, 2)
	}))
}

// TestSplitsHandlePos tests moving the handles of tiles and of sub-splits.
func TestSplitsHandlePos(t *testing.T) {
	b, sp := makeSplits(3, 40, 20)
	sp.updateSplits() // (as it is before the first render)
	sp.setHandlePos(0, 0.5)
	assertSplits(t, sp, 0.5, 0.25, 0.25)
	sp.setHandlePos(1, 0.75)
	assertSplits(t, sp, 0.5, 0.25, 0.25)
	sp.setHandlePos(1, 1)
	assertSplits(t, sp, 0.5, 0.5, 0)
	sp.setHandlePos(1, 0.8)
	assertSplits(t, sp, 0.5, 0.3, 0.2)
	sp.setHandlePos(0, -1)
	assertSplits(t, sp, 0, 0.6, 0.4)

	// a change of a handle sets the position from where it is
	AssertRender(t, b, "splits/handle-pos", Locked(b, func() {
		h := sp.Parts.Child(0).(*Handle)
		h.SendChange()
		assert.InDelta(t, 0, sp.Splits()[0], 0.05)
	}))
}

// TestSplitsTilesOps tests setting the splits, handles, and split indexes
// of each kind of tile, and collapsing the tiles of more children than
// tiles with the number keys.
func TestSplitsTilesOps(t *testing.T) {
	_, sp := makeSplits(12, 40, 20)
	sp.SetTiles(TileSplit, TileFirstLong, TileSecondLong, TilePlus)
	sp.updateSplits()
	sum := func(s []float32) float32 {
		var t float32
		for _, v := range s {
			t += v
		}
		return t
	}
	for _, ss := range sp.SubSplits {
		assert.InDelta(t, 1, sum(ss[:2]), 0.001)
	}

	// handles: 3 between the tiles, then 1, 2, 2, and 3 within them
	for i := range 3 + 1 + 2 + 2 + 3 {
		sp.setHandlePos(i, 0.3)
	}
	assert.InDelta(t, 0.3, sp.TileSplits[0], 0.001)
	assert.InDelta(t, 1, sum(sp.TileSplits), 0.001)
	assert.InDelta(t, 0.3, sp.SubSplits[0][0], 0.001)
	assert.InDelta(t, 0.3, sp.SubSplits[1][2], 0.001)
	assert.InDelta(t, 0.3, sp.SubSplits[3][4], 0.001)

	// split indexes: 4 tiles, then 2, 3, 3, and 4 within them
	for i := range 4 + 2 + 3 + 3 + 4 {
		sp.setSplitIndex(i, 0.2)
	}
	assert.InDelta(t, 1, sum(sp.TileSplits), 0.001)
	for i, ss := range sp.SubSplits {
		assert.InDelta(t, 1, sum(ss[:2]), 0.001, i)
	}

	sp.SetSplits(0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.1, 0.2, 0.3)
	assert.Len(t, sp.Splits(), 12)
	sendSplitsKey(sp, 4) // the last of the 4 tiles
	assert.Equal(t, float32(0), sp.TileSplits[3])
	sendSplitsKey(sp, 4)
	assert.Greater(t, sp.TileSplits[3], float32(0), "restored")
	sendSplitsKey(sp, 5) // more than the tiles, which used to panic
	assert.Len(t, sp.TileSplits, 4)
}
