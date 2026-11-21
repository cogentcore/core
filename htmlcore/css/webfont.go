// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

// SplitTopLevel splits the given string at the given separator where it is
// not in parentheses or quotes (as a list of sources with url()s).
func SplitTopLevel(s string, sep byte) []string {
	var parts []string
	for {
		before, after, found := CutTopLevel(s, sep)
		parts = append(parts, before)
		if !found {
			return parts
		}
		s = after
	}
}
