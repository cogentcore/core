// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nptime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTime(t *testing.T) {
	tn := time.Now()
	tp := Time{}
	tp.SetTime(tn)
	tnr := tp.Time()

	if !tn.Equal(tnr) {
		t.Errorf("time was not reconstructed properly: %v vs. %v\n", tn, tnr)
	}
}

func TestTimeZeroNow(t *testing.T) {
	var nt Time
	assert.True(t, nt.IsZero())
	before := time.Now()
	nt.Now()
	assert.False(t, nt.IsZero())
	assert.False(t, nt.Time().Before(before.Truncate(time.Second)))
	tt := time.Date(2026, 1, 2, 3, 4, 5, 6, time.UTC)
	nt.SetTime(tt)
	assert.True(t, tt.Equal(nt.Time()), "the same instant, to the nanosecond")
}
