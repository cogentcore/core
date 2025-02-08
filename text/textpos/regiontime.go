// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpos

import (
	"fmt"
	"strings"
	"time"

	"cogentcore.org/core/base/nptime"
)

// RegionTime is a [Region] that has Time stamp for when the region was created
// as valid positions into the lines source.
type RegionTime struct {
	Region

	// Time when region was set: needed for updating locations in the text based
	// on time stamp (using efficient non-pointer time).
	Time nptime.Time
}

// TimeNow grabs the current time as the edit time.
func (tr *RegionTime) TimeNow() {
	tr.Time.Now()
}

// NewRegionTime creates a new text region using separate line and char
// values for start and end, and also sets the time stamp to now.
func NewRegionTime(stLn, stCh, edLn, edCh int) RegionTime {
	tr := RegionTime{Region: NewRegion(stLn, stCh, edLn, edCh)}
	tr.TimeNow()
	return tr
}

// NewRegionPosTime creates a new text region using position values
// and also sets the time stamp to now.
func NewRegionPosTime(st, ed Pos) RegionTime {
	tr := RegionTime{Region: NewRegionPos(st, ed)}
	tr.TimeNow()
	return tr
}

// NewRegionLenTime makes a new Region from a starting point and a length
// along same line, and sets the time stamp to now.
func NewRegionLenTime(start Pos, len int) RegionTime {
	tr := RegionTime{Region: NewRegionLen(start, len)}
	tr.TimeNow()
	return tr
}

// IsAfterTime reports if this region's time stamp is after given time value
// if region Time stamp has not been set, it always returns true
func (tr *RegionTime) IsAfterTime(t time.Time) bool {
	if tr.Time.IsZero() {
		return true
	}
	return tr.Time.Time().After(t)
}

// Ago returns how long ago this Region's time stamp is relative
// to given time.
func (tr *RegionTime) Ago(t time.Time) time.Duration {
	return t.Sub(tr.Time.Time())
}

// Age returns the time interval from [time.Now]
func (tr *RegionTime) Age() time.Duration {
	return tr.Ago(time.Now())
}

// Since returns the time interval between
// this Region's time stamp and that of the given earlier region's stamp.
func (tr *RegionTime) Since(earlier *RegionTime) time.Duration {
	return earlier.Ago(tr.Time.Time())
}

// FromString decodes text region from a string representation of form:
// [#]LxxCxx-LxxCxx. Used in e.g., URL links -- returns true if successful
func (tr *RegionTime) FromString(link string) bool {
	link = strings.TrimPrefix(link, "#")
	fmt.Sscanf(link, "L%dC%d-L%dC%d", &tr.Start.Line, &tr.Start.Char, &tr.End.Line, &tr.End.Char)
	tr.Start.Line--
	tr.Start.Char--
	tr.End.Line--
	tr.End.Char--
	return true
}
