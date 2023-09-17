// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/mattn/go-shellwords"
	"goki.dev/gti"
)

// ParseDirective parses and returns a comment directive from
// the given comment string. The returned directive will be nil
// if there is no directive contained in the given comment.
// Directives are of the following form (the slashes are optional):
//
//	//tool:directive args...
func ParseDirective(comment string) (*gti.Directive, error) {
	comment = strings.TrimPrefix(comment, "//")
	rs := []rune(comment)
	if len(rs) == 0 || unicode.IsSpace(rs[0]) { // directives must not have whitespace as their first character
		return nil, nil
	}
	before, after, found := strings.Cut(comment, ":")
	if !found {
		return nil, nil
	}
	directive := &gti.Directive{}
	directive.Tool = before
	args, err := shellwords.Parse(after)
	if err != nil {
		return nil, fmt.Errorf("error parsing args %w", err)
	}
	directive.Args = args
	if len(args) > 0 {
		directive.Directive = directive.Args[0]
		directive.Args = directive.Args[1:]
	}
	return directive, nil
}
