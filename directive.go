// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import "strings"

// ParseDirective parses a comment directive and returns
// the tool, the directive, and true if the comment is
// a directive, and "", "", false if it is not. Directives
// are of the following form (the slashes are optional):
//
//	//tool:directive...
func ParseDirective(comment string) (tool, directive string, has bool) {
	comment = strings.TrimPrefix(comment, "//")
	before, after, found := strings.Cut(comment, ":")
	if !found {
		return
	}
	has = true
	tool = before
	fields := strings.Fields(after)
	if len(fields) != 0 {
		directive = fields[0]
	}
	return
}
