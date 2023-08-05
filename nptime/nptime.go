// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package nptime provides a non-pointer version of the time.Time struct
that does not have the location pointer information that time.Time has,
which is more efficient from a memory management perspective, in cases
where you have a lot of time values being kept: https://segment.com/blog/allocation-efficiency-in-high-performance-go-services/
*/
package nptime

import "time"

// Time represents the value of time.Time without using any pointers for the
// location information, so it is more memory efficient when lots of time
// values are being stored.
type Time struct {

	// time.Time.Unix() seconds since 1970
	Sec int64 `desc:"time.Time.Unix() seconds since 1970"`

	// time.Time.Nanosecond() -- nanosecond offset within second, *not* UnixNano()
	NSec uint32 `desc:"time.Time.Nanosecond() -- nanosecond offset within second, *not* UnixNano()"`
}

// TimeZero is the uninitialized zero time value -- use to check whether
// time has been set or not
var TimeZero Time

// IsZero returns true if the time is zero and has not been initialized
func (t Time) IsZero() bool {
	return t == TimeZero
}

// Time returns the time.Time value for this nptime.Time value
func (t Time) Time() time.Time {
	return time.Unix(t.Sec, int64(t.NSec))
}

// SetTime sets the nptime.Time value based on the time.Time value
func (t *Time) SetTime(tt time.Time) {
	t.Sec = tt.Unix()
	t.NSec = uint32(tt.Nanosecond())
}

// Now sets the time value to time.Now()
func (t *Time) Now() {
	t.SetTime(time.Now())
}
