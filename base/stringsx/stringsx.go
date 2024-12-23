// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package stringsx provides additional string functions
// beyond those in the standard [strings] package.
package stringsx

import (
	"bytes"
	"slices"
	"strings"
)

// TrimCR returns the string without any trailing \r carriage return
func TrimCR(s string) string {
	n := len(s)
	if n == 0 {
		return s
	}
	if s[n-1] == '\r' {
		return s[:n-1]
	}
	return s
}

// ByteTrimCR returns the byte string without any trailing \r carriage return
func ByteTrimCR(s []byte) []byte {
	n := len(s)
	if n == 0 {
		return s
	}
	if s[n-1] == '\r' {
		return s[:n-1]
	}
	return s
}

// SplitLines is a windows-safe version of [strings.Split](s, "\n")
// that removes any trailing \r carriage returns from the split lines.
func SplitLines(s string) []string {
	ls := strings.Split(s, "\n")
	for i, l := range ls {
		ls[i] = TrimCR(l)
	}
	return ls
}

// ByteSplitLines is a windows-safe version of [bytes.Split](s, "\n")
// that removes any trailing \r carriage returns from the split lines.
func ByteSplitLines(s []byte) [][]byte {
	ls := bytes.Split(s, []byte("\n"))
	for i, l := range ls {
		ls[i] = ByteTrimCR(l)
	}
	return ls
}

// InsertFirstUnique inserts the given string at the start of the given string slice
// while keeping the overall length to the given max value. If the item is already on
// the list, then it is moved to the top and not re-added (unique items only). This is
// useful for a list of recent items.
func InsertFirstUnique(strs *[]string, str string, max int) {
	if *strs == nil {
		*strs = make([]string, 0, max)
	}
	sz := len(*strs)
	if sz > max {
		*strs = (*strs)[:max]
	}
	for i, s := range *strs {
		if s == str {
			if i == 0 {
				return
			}
			copy((*strs)[1:i+1], (*strs)[0:i])
			(*strs)[0] = str
			return
		}
	}
	if sz >= max {
		copy((*strs)[1:max], (*strs)[0:max-1])
		(*strs)[0] = str
	} else {
		*strs = append(*strs, "")
		if sz > 0 {
			copy((*strs)[1:], (*strs)[0:sz])
		}
		(*strs)[0] = str
	}
}

// UniqueList removes duplicates from given string list,
// preserving the order.
func UniqueList(strs []string) []string {
	n := len(strs)
	for i := n - 1; i >= 0; i-- {
		p := strs[i]
		for j, s := range strs {
			if p == s && i != j {
				strs = slices.Delete(strs, i, i+1)
			}
		}
	}
	return strs
}
