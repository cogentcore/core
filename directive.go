// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"strings"

	"github.com/mattn/go-shellwords"
)

// ParseDirective parses a comment directive and returns
// the tool, the directive, and true if the comment is
// a directive, and "", "", false if it is not. Directives
// are of the following form (the slashes are optional):
//
//	//tool:directive arg0 -arg1=go ...
func ParseDirective(comment string) (tool, directive string, args []string, has bool, err error) {
	comment = strings.TrimPrefix(comment, "//")
	before, after, found := strings.Cut(comment, ":")
	if !found {
		return
	}
	has = true
	tool = before
	args, err = shellwords.Parse(after)
	if err != nil {
		err = fmt.Errorf("error parsing args %w", err)
		return
	}
	if len(args) > 0 {
		directive = args[0]
		args = args[1:]
	}
	return
}
