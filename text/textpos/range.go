// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpos

// Range defines a range with a start and end index, where end is typically
// exclusive, as in standard slice indexing and for loop conventions.
type Range struct {
	// St is the starting index of the range.
	Start int

	// Ed is the ending index of the range.
	End int
}

// Len returns the length of the range: End - Start.
func (r Range) Len() int {
	return r.End - r.Start
}

// Contains returns true if range contains given index.
func (r Range) Contains(i int) bool {
	return i >= r.Start && i < r.End
}

// Intersect returns the intersection of two ranges.
// If they do not overlap, then the Start and End will be -1
func (r Range) Intersect(o Range) Range {
	o.Start = max(o.Start, r.Start)
	o.End = min(o.End, r.End)
	if o.Len() <= 0 {
		return Range{-1, -1}
	}
	return o
}
