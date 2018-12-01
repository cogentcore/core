// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"fmt"
	"strings"

	"github.com/goki/ki/kit"
	"github.com/goki/pi/lex"
)

// TraceOpts provides options for debugging / monitoring the rule matching and execution process
type TraceOpts struct {
	On        bool     `desc:"perform tracing"`
	Rules     string   `width:"60" desc:"trace specific named rules here (space separated) -- if blank, then all rules are traced"`
	Match     bool     `desc:"trace matches -- why a rule matches"`
	NoMatch   bool     `desc:"trace non-matches -- why a rule doesn't match (can be a lot of info)"`
	Run       bool     `desc:"trace progress runing through each of the sub-rules when a rule has matched and is 'running'"`
	ScopeSrc  bool     `desc:"if true, shows the full scope source for every trace statement"`
	RulesList []string `view:"-" json:"-" xml:"-" desc:"list of rules"`
}

// parse.Trace controls the tracing options for debugging / monitoring the rule matching and execution process
var Trace TraceOpts

// Init intializes tracer after any changes
func (pt *TraceOpts) Init() {
	if pt.Rules == "" {
		pt.RulesList = nil
	} else {
		pt.RulesList = strings.Split(pt.Rules, " ")
	}
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
	case NoMatch:
		if !pt.NoMatch {
			return false
		}
	case Run:
		if !pt.Run {
			return false
		}
	}
	tokSrc := pos.String() + `"` + string(ps.Src.TokenSrc(pos)) + `"`
	plev := ast.ParentLevel(ps.Ast)
	ind := ""
	for i := 0; i < plev; i++ {
		ind += "\t"
	}
	fmt.Printf("%v%v:\t %v\t %v\t tok: %v\t scope: %v\t ast: %v\n", ind, pr.Name(), step, msg, tokSrc, scope, ast.PathUnique())
	if pt.ScopeSrc {
		scopeSrc := ps.Src.TokenRegSrc(scope)
		fmt.Printf("%v\t%v\n", ind, scopeSrc)
	}
	return true
}

// Steps are the different steps of the parsing processing
type Steps int

//go:generate stringer -type=Steps

var KiT_Steps = kit.Enums.AddEnum(StepsN, false, nil)

func (ev Steps) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Steps) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// The parsing steps
const (
	// Match happens when a rule matches
	Match Steps = iota

	// NoMatch is when the rule fails to match
	NoMatch

	// Run is when the rule is running and iterating through its sub-rules
	Run

	StepsN
)
