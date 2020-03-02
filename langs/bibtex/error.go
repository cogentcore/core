// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copied and only lightly modified from:
// https://github.com/nickng/bibtex
// Licenced under an Apache-2.0 licence
// and presumably Copyright (c) 2017 by Nick Ng

package bibtex

import (
	"errors"
	"fmt"
)

var (
	// ErrUnexpectedAtsign is an error for unexpected @ in {}.
	ErrUnexpectedAtsign = errors.New("Unexpected @ sign")
	// ErrUnknownStringVar is an error for looking up undefined string var.
	ErrUnknownStringVar = errors.New("Unknown string variable")
)

// ErrParse is a parse error.
type ErrParse struct {
	Pos TokenPos
	Err string // Error string returned from parser.
}

func (e *ErrParse) Error() string {
	return fmt.Sprintf("Parse failed at %s: %s", e.Pos, e.Err)
}
