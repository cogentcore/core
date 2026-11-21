// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

// SplitTopLevelSpaces splits the given value at the spaces that are not in
// parentheses (as "calc(1rem + 4px) auto").
func SplitTopLevelSpaces(v string) []string {
	var parts []string
	depth, start := 0, -1
	for i, r := range v {
		switch {
		case r == '(':
			depth++
		case r == ')':
			depth--
		case r == ' ' && depth == 0:
			if start >= 0 {
				parts = append(parts, v[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		parts = append(parts, v[start:])
	}
	return parts
}
