// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

var end = "\x00"
var endChar byte = '\x00'

func CString(s string) string {
	if len(s) == 0 {
		return end
	}
	if s[len(s)-1] != endChar {
		return s + end
	}
	return s
}

func CStrings(list []string) []string {
	for i := range list {
		list[i] = CString(list[i])
	}
	return list
}
