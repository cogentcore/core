// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package wintoast

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToasts tests scheduling toasts, which replace the ones scheduled
// before, and showing one (not in short mode, as it pops up).
func TestToasts(t *testing.T) {
	nt, err := New("CogentCore.WinToastTest", "Cogent Core toast test")
	require.NoError(t, err)
	at := time.Now().Add(time.Hour)
	require.NoError(t, nt.Sync([]*Toast{{ID: "a", Title: "A <&>", Body: "body", Time: at}, {ID: "b", Title: "B", Time: at.Add(time.Minute)}}))
	n, err := nt.NumScheduled()
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	require.NoError(t, nt.Sync([]*Toast{{ID: "c", Title: "C", Time: at}}))
	n, _ = nt.NumScheduled()
	assert.Equal(t, 1, n, "replaced")
	require.NoError(t, nt.Sync(nil))
	n, _ = nt.NumScheduled()
	assert.Equal(t, 0, n, "canceled")
	assert.Equal(t, 16, len(shortID("a long ID of a notification")))
	if !testing.Short() {
		assert.NoError(t, nt.Show(&Toast{ID: "now", Title: "Toast test", Body: "Shown now"}))
	}
}
