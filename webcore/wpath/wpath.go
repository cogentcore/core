// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package wpath handles webcore paths
package wpath

import (
	"strings"
	"unicode"
)

// Format formats the given path into a correct webcore path
// by removing all `{digit(s)}-` prefixes at the start of path
// segments, which are used for ordering files and folders and
// thus should not be displayed.
func Format(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if !strings.Contains(part, "-") {
			continue
		}
		parts[i] = strings.TrimLeftFunc(part, func(r rune) bool {
			return unicode.IsDigit(r) || r == '-'
		})
	}
	return strings.Join(parts, "/")
}
