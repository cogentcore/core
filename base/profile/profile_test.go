// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package profile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProfile tests timing named functions, which only happens while
// profiling, and reporting their times.
func TestProfile(t *testing.T) {
	assert.Nil(t, Start(), "not profiling")
	Report(time.Millisecond)
	Profiling = true
	defer func() { Profiling = false; Reset() }()
	Reset()

	for range 2 {
		pr := Start("a")
		require.NotNil(t, pr)
		assert.Nil(t, Start("a"), "already timing")
		time.Sleep(time.Millisecond)
		pr.End()
	}
	StartName("named", "x").End()
	var np *Profile
	np.End()

	p := TheProfiler.Profiles["profile.TestProfile-a"]
	require.NotNil(t, p)
	assert.Equal(t, int64(2), p.N)
	assert.GreaterOrEqual(t, p.Total, 2*time.Millisecond)
	assert.Equal(t, int64(1), TheProfiler.Profiles["named-x"].N)
	Report(time.Millisecond)
	Reset()
	assert.Empty(t, TheProfiler.Profiles)
}
