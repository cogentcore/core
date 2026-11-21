// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"strings"
)

// CutTopLevel cuts s around the first sep that is not in parentheses.
func CutTopLevel(s string, sep byte) (before, after string, found bool) {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case sep:
			if depth == 0 {
				return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]), true
			}
		}
	}
	return s, "", false
}
