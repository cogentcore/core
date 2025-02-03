// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpos

// Region is a contiguous region within the source file,
// defined by start and end [Pos] positions.
type Region struct {
	// starting position of region
	St Pos
	// ending position of region
	Ed Pos
}

// IsNil checks if the region is empty, because the start is after or equal to the end.
func (tr Region) IsNil() bool {
	return !tr.St.IsLess(tr.Ed)
}

// Contains returns true if region contains position
func (tr Region) Contains(ps Pos) bool {
	return ps.IsLess(tr.Ed) && (tr.St == ps || tr.St.IsLess(ps))
}
