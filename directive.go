// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"strings"

	"github.com/mattn/go-shellwords"
	"goki.dev/gti"
)

// ParseDirective parses a comment directive and returns
// the tool, the directive, and the arguments if the comment is
// a directive, and zero values for everything if not. Directives
// are of the following form (the slashes are optional):
//
//	//tool:directive arg0 -arg1=go ...
func ParseDirective(comment string) (directive gti.Directive, has bool, err error) {
	comment = strings.TrimPrefix(comment, "//")
	before, after, found := strings.Cut(comment, ":")
	if !found {
		return
	}
	has = true
	directive.Tool = before
	directive.Args, err = shellwords.Parse(after)
	if err != nil {
		err = fmt.Errorf("error parsing args %w", err)
		return
	}
	if len(directive.Args) > 0 {
		directive.Directive = directive.Args[0]
		directive.Args = directive.Args[1:]
	}
	return
}
