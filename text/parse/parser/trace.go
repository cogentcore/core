// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"
	"os"
	"strings"

	"cogentcore.org/core/text/textpos"
)

// TraceOptions provides options for debugging / monitoring the rule matching and execution process
type TraceOptions struct {

	// perform tracing
	On bool

	// trace specific named rules here (space separated) -- if blank, then all rules are traced
	Rules string `width:"50"`

	// trace full rule matches -- when a rule fully matches
	Match bool

	// trace sub-rule matches -- when the parts of each rule match
	SubMatch bool

	// trace sub-rule non-matches -- why a rule doesn't match -- which terminates the matching process at first non-match (can be a lot of info)
	NoMatch bool

	// trace progress running through each of the sub-rules when a rule has matched and is 'running'
	Run bool

	// trace actions performed by running rules
	RunAct bool

	// if true, shows the full scope source for every trace statement
	ScopeSrc bool

	// for the ParseOut display, whether to display the full stack of rules at each position, or just the deepest one
	FullStackOut bool

	// list of rules
	RulesList []string `display:"-" json:"-" xml:"-"`

	// trace output is written here, connected via os.Pipe to OutRead
	OutWrite *os.File `display:"-" json:"-" xml:"-"`

	// trace output is read here; can connect this using [textcore.OutputBuffer] to monitor tracing output
	OutRead *os.File `display:"-" json:"-" xml:"-"`
}

// Init intializes tracer after any changes -- opens pipe if not already open
func (pt *TraceOptions) Init() {
	if pt.Rules == "" {
		pt.RulesList = nil
	} else {
		pt.RulesList = strings.Split(pt.Rules, " ")
	}
}

// FullOn sets all options on
func (pt *TraceOptions) FullOn() {
	pt.On = true
	pt.Match = true
	pt.SubMatch = true
	pt.NoMatch = true
	pt.Run = true
	pt.RunAct = true
	pt.ScopeSrc = true
}

// PipeOut sets output to a pipe for monitoring (OutWrite -> OutRead)
func (pt *TraceOptions) PipeOut() {
	if pt.OutWrite == nil {
		pt.OutRead, pt.OutWrite, _ = os.Pipe() // seriously, does this ever fail?
	}
}

// Stdout sets [TraceOptions.OutWrite] to [os.Stdout]
func (pt *TraceOptions) Stdout() {
	pt.OutWrite = os.Stdout
}

// CheckRule checks if given rule should be traced
func (pt *TraceOptions) CheckRule(rule string) bool {
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
func (pt *TraceOptions) Out(ps *State, pr *Rule, step Steps, pos textpos.Pos, scope textpos.Region, ast *AST, msg string) bool {
	if !pt.On {
		return false
	}
	if !pt.CheckRule(pr.Name) {
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
	plev := ast.ParentLevel(ps.AST)
	ind := ""
	for i := 0; i < plev; i++ {
		ind += "\t"
	}
	fmt.Fprintf(pt.OutWrite, "%v%v:\t %v\t %v\t tok: %v\t scope: %v\t ast: %v\n", ind, pr.Name, step, msg, tokSrc, scope, ast.Path())
	if pt.ScopeSrc {
		scopeSrc := ps.Src.TokenRegSrc(scope)
		fmt.Fprintf(pt.OutWrite, "%v\t%v\n", ind, scopeSrc)
	}
	return true
}

// CopyOpts copies just the options
func (pt *TraceOptions) CopyOpts(ot *TraceOptions) {
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
type Steps int32 //enums:enum

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
