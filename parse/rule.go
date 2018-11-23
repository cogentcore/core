// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parse does the parsing stage after lexing
package parse

import (
	"reflect"

	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// Parser is the interface type for parsers -- likely not necessary except is essential
// for defining the BaseIface for gui in making new nodes
type Parser interface {
	ki.Ki
	// Parse tries to apply rule to given input state, returns true if matched, false if not
	Parse() bool
}

// parse.Rule operates on the text input to produce the lexical tokens
// it is assembled into a lexical grammar structure to perform lexing
//
// Parsing is done line-by-line -- you must push and pop states to
// coordinate across multiple lines, e.g., for multi-line comments
//
// In general it is best to keep lexing as simple as possible and
// leave the more complex things for the parsing step.
type Rule struct {
	ki.Node
	Off       int    `desc:"offset into the input to look for a match: 0 = current char, 1 = next one, etc"`
	String    string `desc:"if action is LexMatch, this is the string we match"`
	PushState string `desc:"the state to push if our action is PushState -- note that State matching is on String, not this value"`
}

var KiT_Rule = kit.Types.AddType(&Rule{}, RuleProps)

func (pr *Rule) BaseIface() reflect.Type {
	return reflect.TypeOf((*Parser)(nil)).Elem()
}

// Parse tries to apply rule to given input state, returns true if matched, false if not
func (pr *Rule) Parse() bool {
	return false
}

var RuleProps = ki.Props{
	// "CallMethods": ki.PropSlice{
	// 	{"SaveAs", ki.Props{
	// 		"Args": ki.PropSlice{
	// 			{"File Name", ki.Props{
	// 				"default-field": "Filename",
	// 			}},
	// 		},
	// 	}},
	// },
}
