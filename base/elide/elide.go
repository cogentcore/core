// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package elide provides basic text eliding functions.
package elide

import "strings"

// End elides from the end of the string if it is longer than given
// size parameter.  The resulting string will not exceed sz in length,
// with space reserved for … at the end.
func End(s string, sz int) string {
	n := len(s)
	if n <= sz {
		return s
	}
	return s[:sz-1] + "…"
}

// Middle elides from the middle of the string if it is longer than given
// size parameter.  The resulting string will not exceed sz in length,
// with space reserved for … in the middle
func Middle(s string, sz int) string {
	n := len(s)
	if n <= sz {
		return s
	}
	en := sz - 1
	mid := en / 2
	rest := en - mid
	return s[:mid] + "…" + s[n-rest:]
}

// AppName elides the given app name to be twelve characters or less
// by removing word(s) from the middle of the string if necessary and possible.
func AppName(s string) string {
	if len(s) <= 12 {
		return s
	}
	words := strings.Fields(s)
	if len(words) < 3 {
		return s
	}
	return words[0] + " " + words[len(words)-1]
}
