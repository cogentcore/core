// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpos

// Range defines a range with a start and end index, where end is typically
// inclusive, as in standard slice indexing and for loop conventions.
type Range struct {
	// St is the starting index of the range.
	Start int

	// Ed is the ending index of the range.
	End int
}

// Len returns the length of the range: Ed - St.
func (r Range) Len() int {
	return r.End - r.Start
}

// Contains returns true if range contains given index.
func (r Range) Contains(i int) bool {
	return i >= r.Start && i < r.End
}
