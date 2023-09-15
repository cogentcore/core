// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bools

// ToString converts a bool to "true" or "false" string
func ToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// FromString converts string to a bool, "true" = true, else false
func FromString(v string) bool {
	if v == "true" {
		return true
	}
	return false
}
