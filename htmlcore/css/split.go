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
