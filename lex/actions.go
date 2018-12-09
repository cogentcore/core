// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"github.com/goki/ki/kit"
)

// Actions are lexing actions to perform
type Actions int

//go:generate stringer -type=Actions

var KiT_Actions = kit.Enums.AddEnum(ActionsN, false, nil)

func (ev Actions) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Actions) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// The lexical acts
const (
	// Next means advance input position to the next character(s) after the matched characters
	Next Actions = iota

	// Name means read in an entire name, which is letters, _ and digits after first letter
	// position will be advanced to just after
	Name

	// Number means read in an entire number -- the token type will automatically be
	// set to the actual type of number that was read in, and position advanced to just after
	Number

	// Quoted means read in an entire string enclosed in quote delimeter
	// that is present at current position, with proper skipping of escaped.
	// Position advanced to just after
	Quoted

	// QuotedRaw means read in an entire string enclosed in quote delimeter
	// that is present at start position, with proper skipping of escaped.
	// Position advanced to just after.
	// Raw version supports multi-line and includes CR etc at end of lines (e.g., back-tick
	// in various languages)
	QuotedRaw

	// EOL means read till the end of the line (e.g., for single-line comments)
	EOL

	// PushState means push the given state value onto the state stack
	PushState

	// PopState means pop given state value off the state stack
	PopState

	// SetGuestLex means install the Name (must be a prior action) as the guest
	// lexer -- it will take over lexing until PopGuestLex is called
	SetGuestLex

	// PopGuestLex removes the current guest lexer and returns to the original
	// language lexer
	PopGuestLex

	ActionsN
)
