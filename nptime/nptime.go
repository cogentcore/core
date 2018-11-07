// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/* package nptime provides a non-pointer version of the time.Time struct
that does not have the location pointer information that time.Time has,
which is more efficient from a memory management perspective, in cases
where you have a lot of time values being kept: https://segment.com/blog/allocation-efficiency-in-high-performance-go-services/
*/
package nptime

import "time"

type Time struct {
	Sec  int64
	NSec int64
}

// Time returns the time.Time value for this nptime.Time value
func (t Time) Time() time.Time {
	return time.Unix(t.Sec, t.NSec)
}

// SetTime sets the nptime.Time value based on the time.Time value
func (t *Time) SetTime(tt time.Time) {
	t.Sec = tt.Unix()
	t.NSec = tt.UnixNano()
}
