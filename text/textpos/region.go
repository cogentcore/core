// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpos

// Region is a contiguous region within the source file,
// defined by start and end [Pos] positions.
type Region struct {
	// starting position of region
	Start Pos
	// ending position of region
	End Pos
}

// IsNil checks if the region is empty, because the start is after or equal to the end.
func (tr Region) IsNil() bool {
	return !tr.Start.IsLess(tr.End)
}

// Contains returns true if region contains position
func (tr Region) Contains(ps Pos) bool {
	return ps.IsLess(tr.End) && (tr.Start == ps || tr.Start.IsLess(ps))
}
