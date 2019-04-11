// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

var end = "\x00"
var endChar byte = '\x00'

// CString returns a null-terminated string if not already so
func CString(s string) string {
	sz := len(s)
	if sz == 0 {
		return end
	}
	if s[sz-1] != endChar {
		return s + end
	}
	return s
}

// CStrings returns null-terminated strings if not already so
func CStrings(list []string) []string {
	for i := range list {
		list[i] = CString(list[i])
	}
	return list
}

// GoString returns a non-null-terminated string if not already so
func GoString(s string) string {
	sz := len(s)
	if sz == 0 {
		return s
	}
	if s[sz-1] == endChar {
		return s[0 : sz-1]
	}
	return s
}
