// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
	"time"
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
