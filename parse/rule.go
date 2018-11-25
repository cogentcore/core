// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parse does the parsing stage after lexing
package parse

import (
	"reflect"

	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/token"
)

// Parser is the interface type for parsers -- likely not necessary except is essential
// for defining the BaseIface for gui in making new nodes
type Parser interface {
	ki.Ki

	// Compile compiles string rules into their runnable elements
	Compile() bool

	// Validate checks for any errors in the rules and issues warnings,
	// returns true if valid (no err) and false if invalid (errs)
	Validate() bool

	// Parse tries to apply rule to given input state, returns true if matched, false if not
	Parse() bool

	// AsParseRule returns object as a parse.Rule
	AsParseRule() *Rule
}

// RuleEl is an element of a parsing rule -- either a pointer to another rule or a token
type RuleEl struct {
	Rule  *Rule        `desc:"rule -- nil if token"`
	Token token.Tokens `desc:"token, None if rule"`
}

func (re RuleEl) IsRule() bool {
	return re.Rule != nil
}

func (re RuleEl) IsToken() bool {
	return re.Rule == nil
}

// RuleList is a list (slice) of rule elements
type RuleList []RuleEl

// todo: need a map of all rules to make it fast

// parse.Rule operates on the lexically-tokenized input, not the raw source.
//
// The overall strategy is very pragmatic and based on the current known form of
// most languages, which are organized around a sequence of statements having
// a clear scoping defined by the EOS (end of statement), which is identified
// in a first pass through tokenized output using
//
// Each rule is triggered by a single key token (KeyTok) which is the first distinctive
// concrete token associated with this rule.  Each rule also has a well-defined scope
// (start, end) which is either given explicitly by the rule in the case of statements,
// by concluding with `EOS`, or by higher-level parse matches that have determined
// the scope for this element (as represented in the associated Ast).
//
// There are two different styles of rules: parents with multiple children
// and the children that specify various alternative forms of the parent category
type Rule struct {
	ki.Node
	Desc      string       `desc:"description / comments about this rule"`
	Rule      string       `desc:"the rule as a space-separated list of rule names and token(s) -- use single quotes around tokens -- first one becomes the key token"`
	Rules     RuleList     `desc:"rule elements compiled from Rule string"`
	KeyTok    token.Tokens `desc:"the token value that this rule matches, if Match = Token"`
	Keyword   string       `desc:"if the token is Keyword, this is the specific token we match"`
	PushState string       `desc:"the state to push if our action is PushState -- note that State matching is on String, not this value"`
}

var KiT_Rule = kit.Types.AddType(&Rule{}, RuleProps)

func (pr *Rule) BaseIface() reflect.Type {
	return reflect.TypeOf((*Parser)(nil)).Elem()
}

func (pr *Rule) AsParseRule() *Rule {
	return pr.This().Embed(KiT_Rule).(*Rule)
}

// Validate checks for any errors in the rules and issues warnings,
// returns true if valid (no err) and false if invalid (errs)
func (pr *Rule) Validate() bool {
	return true
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
