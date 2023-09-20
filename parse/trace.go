// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"fmt"
	"os"
	"strings"

	"goki.dev/pi/v2/lex"
)

// TraceOpts provides options for debugging / monitoring the rule matching and execution process
type TraceOpts struct {

	// perform tracing
	On bool `desc:"perform tracing"`

	// trace specific named rules here (space separated) -- if blank, then all rules are traced
	Rules string `width:"50" desc:"trace specific named rules here (space separated) -- if blank, then all rules are traced"`

	// trace full rule matches -- when a rule fully matches
	Match bool `desc:"trace full rule matches -- when a rule fully matches"`

	// trace sub-rule matches -- when the parts of each rule match
	SubMatch bool `desc:"trace sub-rule matches -- when the parts of each rule match"`

	// trace sub-rule non-matches -- why a rule doesn't match -- which terminates the matching process at first non-match (can be a lot of info)
	NoMatch bool `desc:"trace sub-rule non-matches -- why a rule doesn't match -- which terminates the matching process at first non-match (can be a lot of info)"`

	// trace progress running through each of the sub-rules when a rule has matched and is 'running'
	Run bool `desc:"trace progress running through each of the sub-rules when a rule has matched and is 'running'"`

	// trace actions performed by running rules
	RunAct bool `desc:"trace actions performed by running rules"`

	// if true, shows the full scope source for every trace statement
	ScopeSrc bool `desc:"if true, shows the full scope source for every trace statement"`

	// for the ParseOut display, whether to display the full stack of rules at each position, or just the deepest one
	FullStackOut bool `desc:"for the ParseOut display, whether to display the full stack of rules at each position, or just the deepest one"`

	// [view: -] list of rules
	RulesList []string `view:"-" json:"-" xml:"-" desc:"list of rules"`

	// [view: -] trace output is written here, connected via os.Pipe to OutRead
	OutWrite *os.File `view:"-" json:"-" xml:"-" desc:"trace output is written here, connected via os.Pipe to OutRead"`

	// [view: -] trace output is read here -- can connect this to a TextBuf via giv.OutBuf to monitor tracing output
	OutRead *os.File `view:"-" json:"-" xml:"-" desc:"trace output is read here -- can connect this to a TextBuf via giv.OutBuf to monitor tracing output"`
}

// Init intializes tracer after any changes -- opens pipe if not already open
func (pt *TraceOpts) Init() {
	if pt.Rules == "" {
		pt.RulesList = nil
	} else {
		pt.RulesList = strings.Split(pt.Rules, " ")
	}
}

// FullOn sets all options on
func (pt *TraceOpts) FullOn() {
	pt.On = true
	pt.Match = true
	pt.SubMatch = true
	pt.NoMatch = true
	pt.Run = true
	pt.RunAct = true
	pt.ScopeSrc = true
}

// PipeOut sets output to a pipe for monitoring (OutWrite -> OutRead)
func (pt *TraceOpts) PipeOut() {
	if pt.OutWrite == nil {
		pt.OutRead, pt.OutWrite, _ = os.Pipe() // seriously, does this ever fail?
	}
}

// StdOut sets OutWrite to os.Stdout
func (pt *TraceOpts) StdOut() {
	pt.OutWrite = os.Stdout
}

// CheckRule checks if given rule should be traced
func (pt *TraceOpts) CheckRule(rule string) bool {
	if len(pt.RulesList) == 0 {
		if pt.Rules != "" {
			pt.Init()
			if len(pt.RulesList) == 0 {
				return true
			}
		} else {
			return true
		}
	}
	for _, rn := range pt.RulesList {
		if rn == rule {
			return true
		}
	}
	return false
}

// Out outputs a trace message -- returns true if actually output
func (pt *TraceOpts) Out(ps *State, pr *Rule, step Steps, pos lex.Pos, scope lex.Reg, ast *Ast, msg string) bool {
	if !pt.On {
		return false
	}
	if !pt.CheckRule(pr.Nm) {
		return false
	}
	switch step {
	case Match:
		if !pt.Match {
			return false
		}
	case SubMatch:
		if !pt.SubMatch {
			return false
		}
	case NoMatch:
		if !pt.NoMatch {
			return false
		}
	case Run:
		if !pt.Run {
			return false
		}
	case RunAct:
		if !pt.RunAct {
			return false
		}
	}
	tokSrc := pos.String() + `"` + string(ps.Src.TokenSrc(pos)) + `"`
	plev := ast.ParentLevel(ps.Ast)
	ind := ""
	for i := 0; i < plev; i++ {
		ind += "\t"
	}
	fmt.Fprintf(pt.OutWrite, "%v%v:\t %v\t %v\t tok: %v\t scope: %v\t ast: %v\n", ind, pr.Name(), step, msg, tokSrc, scope, ast.Path())
	if pt.ScopeSrc {
		scopeSrc := ps.Src.TokenRegSrc(scope)
		fmt.Fprintf(pt.OutWrite, "%v\t%v\n", ind, scopeSrc)
	}
	return true
}

// CopyOpts copies just the options
func (pt *TraceOpts) CopyOpts(ot *TraceOpts) {
	pt.On = ot.On
	pt.Rules = ot.Rules
	pt.Match = ot.Match
	pt.SubMatch = ot.SubMatch
	pt.NoMatch = ot.NoMatch
	pt.Run = ot.Run
	pt.RunAct = ot.RunAct
	pt.ScopeSrc = ot.ScopeSrc
}

// Steps are the different steps of the parsing processing
type Steps int //enums:enum

// The parsing steps
const (
	// Match happens when a rule matches
	Match Steps = iota

	// SubMatch is when a sub-rule within a rule matches
	SubMatch

	// NoMatch is when the rule fails to match (recorded at first non-match, which terminates
	// matching process
	NoMatch

	// Run is when the rule is running and iterating through its sub-rules
	Run

	// RunAct is when the rule is running and performing actions
	RunAct
)
