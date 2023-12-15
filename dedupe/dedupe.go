// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	package dedupe implements a de-duplication function for

any comparable slice type, efficiently using a map to
check for duplicates.  The original order of items is preserved.
*/
package dedupe

func DeDupe[T comparable](slc []T) []T {
	unq := make(map[T]struct{})
	rs := make([]T, 0, len(slc))
	for _, it := range slc {
		if _, has := unq[it]; has {
			continue
		}
		unq[it] = struct{}{}
		rs = append(rs, it)
	}
	return rs
}
