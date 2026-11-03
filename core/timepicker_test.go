// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testTime = time.Date(2021, 1, 4, 11, 16, 0, 0, time.UTC)

func TestTimeInput(t *testing.T) {
	b := NewBody()
	NewTimeInput(b).SetTime(testTime)
	b.AssertRender(t, "time-input/basic")
}

func TestTimeInputDate(t *testing.T) {
	b := NewBody()
	NewTimeInput(b).SetTime(testTime).SetDisplayTime(false)
	b.AssertRender(t, "time-input/date")
}

func TestTimeInputTime(t *testing.T) {
	b := NewBody()
	NewTimeInput(b).SetTime(testTime).SetDisplayDate(false)
	b.AssertRender(t, "time-input/time")
}

func TestDurationInput(t *testing.T) {
	b := NewBody()
	NewDurationInput(b).SetDuration(2 * time.Minute)
	b.AssertRender(t, "duration-input/basic")
}

func TestDurationInputReadOnly(t *testing.T) {
	b := NewBody()
	NewDurationInput(b).SetDuration(2 * time.Minute).SetReadOnly(true)
	b.AssertRender(t, "duration-input/read-only")
}

// TestTimeInputZero tests that the zero time is blank, and that clearing
// the date makes the time zero again.
func TestTimeInputZero(t *testing.T) {
	b := NewBody()
	ti := NewTimeInput(b)
	b.AssertRender(t, "time-input/zero", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		date, tm := ti.Child(0).(*TextField), ti.Child(1).(*TextField)
		assert.Equal(t, "", date.Text())
		assert.Equal(t, "", tm.Text())

		date.SetText("3/4/2026").SendChange()
		assert.Equal(t, time.Date(2026, 3, 4, 0, 0, 0, 0, time.Local), ti.Time)
		tm.SetText(time.Date(1, 1, 1, 14, 30, 0, 0, time.UTC).Format(SystemSettings.TimeFormat())).SendChange()
		assert.Equal(t, time.Date(2026, 3, 4, 14, 30, 0, 0, time.Local), ti.Time)

		date.SetText("").SendChange()
		assert.True(t, ti.Time.IsZero(), "cleared")
		ti.Update()
		assert.Equal(t, "", tm.Text())
	})
}
