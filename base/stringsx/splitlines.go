// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package stringsx provides additional string functions
// beyond those in the standard [strings] package.
package stringsx

import "strings"

// SplitLines is a windows-safe version of strings.Split(s, "\n")
// that replaces the carriage return, newline sequence with just newline.
func SplitLines(s string) []string {
	return strings.Split(strings.Replace(s, "\r\n", "\n", -1), "\n")
}
