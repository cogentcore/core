// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package directive implements simple, standardized, and scalable parsing of Go comment directives.
package directive

import (
	"fmt"
	"go/ast"
	"strings"
	"unicode"

	"github.com/mattn/go-shellwords"
)

// Directive represents a comment directive
// that has been parsed or created in code.
type Directive struct {

	// Tool is the name of the tool that
	// the directive is for.
	Tool string

	// Directive is the actual directive
	// string that is placed after the
	// name of the tool and a colon.
	Directive string

	// Args are the arguments
	// passed to the directive
	Args []string
}

// String returns the directive as a formatted string suitable for use in
// code. It includes two slashes (`//`) at the start.
func (d *Directive) String() string {
	if d == nil {
		return "<nil>"
	}
	res := "//" + d.Tool + ":" + d.Directive
	if len(d.Args) > 0 {
		res += " " + strings.Join(d.Args, " ")
	}
	return res
}

// Parse parses the given comment string and returns any [Directive] inside it.
// If no such directive is found, it returns nil. Directives are of the form:
//
//	//tool:directive arg0 key0=value0 arg1 key1=value1
//
// (the two slashes are optional, and the positional and key-value arguments
// can be in any order).
func Parse(comment string) (*Directive, error) {
	comment = strings.TrimPrefix(comment, "//")
	rs := []rune(comment)
	if len(rs) == 0 || unicode.IsSpace(rs[0]) { // directives must not have whitespace as their first character
		return nil, nil
	}
	before, after, found := strings.Cut(comment, ":")
	if !found {
		return nil, nil
	}
	directive := &Directive{}
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

// ParseComment parses the given AST comment
// and returns any [Directive] inside it.
// It is a helper function that calls [Parse]
// on the text of the comment.
func ParseComment(comment *ast.Comment) (*Directive, error) {
	return Parse(comment.Text)
}

// ParseCommentGroup parses the given AST comment
// group and returns a slice of all [Directive]s
// inside it. It is a helper function that calls
// [ParseComment] on each comment in the group.
func ParseCommentGroup(group *ast.CommentGroup) ([]*Directive, error) {
	res := []*Directive{}
	for _, comment := range group.List {
		dir, err := ParseComment(comment)
		if err != nil {
			return nil, err
		}
		if dir != nil {
			res = append(res, dir)
		}
	}
	return res, nil
}
