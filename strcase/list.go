// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package strcase

// FormatList returns a formatted version of the given list of items following these rules:
//   - nil => ""
//   - "Go" => "Go"
//   - "Go", "Python" => "Go and Python"
//   - "Go", "Python", "JavaScript" => "Go, Python, and JavaScript"
//   - "Go", "Python", "JavaScript", "C" => "Go, Python, JavaScript, and C"
func FormatList(items ...string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	}
	res := ""
	for i, match := range items {
		res += match
		if i == len(items)-1 {
			// last one, so do nothing
		} else if i == len(items)-2 {
			res += ", and "
		} else {
			res += ", "
		}
	}
	return res
}
