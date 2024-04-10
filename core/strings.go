// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

// StringsInsertFirstUnique inserts the given string at the start of the given string slice
// while keeping the overall length to the given max value. If the item is already on the list,
// then it is moved to the top and not re-added (unique items only). This is useful for a list
// of recent items.
func StringsInsertFirstUnique(strs *[]string, str string, max int) {
	if *strs == nil {
		*strs = make([]string, 0, max)
	}
	sz := len(*strs)
	if sz > max {
		*strs = (*strs)[:max]
	}
	for i, s := range *strs {
		if s == str {
			if i == 0 {
				return
			}
			copy((*strs)[1:i+1], (*strs)[0:i])
			(*strs)[0] = str
			return
		}
	}
	if sz >= max {
		copy((*strs)[1:max], (*strs)[0:max-1])
		(*strs)[0] = str
	} else {
		*strs = append(*strs, "")
		if sz > 0 {
			copy((*strs)[1:], (*strs)[0:sz])
		}
		(*strs)[0] = str
	}
}
