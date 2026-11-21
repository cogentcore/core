// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

// SplitColors splits the given value into fields at spaces
// that are not inside of parentheses, as in rgb(...) and calc(...).
func SplitColors(v string) []string {
	var fs []string
	depth := 0
	start := -1
	for i, r := range v {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ' ', '\t', '\n':
			if depth == 0 {
				if start >= 0 {
					fs = append(fs, v[start:i])
				}
				start = -1
				continue
			}
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		fs = append(fs, v[start:])
	}
	return fs
}
