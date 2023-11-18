// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package elide provides basic text eliding functions.
*/
package elide

// End elides from the end of the string if it is longer than given
// size parameter.  The resulting string will not exceed sz in length,
// with space reserved for ... at the end.
func End(s string, sz int) string {
	n := len(s)
	if n < sz {
		return s
	}
	return s[:sz-3] + "..."
}

// Middle elides from the middle of the string if it is longer than given
// size parameter.  The resulting string will not exceed sz in length,
// with space reserved for ... in the middle
func Middle(s string, sz int) string {
	n := len(s)
	if n < sz {
		return s
	}
	en := sz - 3
	mid := en / 2
	rest := en - mid
	return s[:mid] + "..." + s[n-rest:]
}
