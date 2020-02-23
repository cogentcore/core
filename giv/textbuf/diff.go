// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ianbruene/go-difflib/difflib"
)

// note: original difflib is: "github.com/pmezard/go-difflib/difflib"

// Diffs are raw differences between text, in terms of lines, reporting a
// sequence of operations that would convert one buffer (a) into the other
// buffer (b).  Each operation is either an 'r' (replace), 'd' (delete), 'i'
// (insert) or 'e' (equal).
type Diffs []difflib.OpCode

// DiffForLine returns the diff record applying to given line, and its index in slice

func (di Diffs) DiffForLine(line int) (int, difflib.OpCode) {
	for i, df := range di {
		if line >= df.I1 && (line < df.I2 || line < df.J2) {
			return i, df
		}
	}
	return -1, difflib.OpCode{}
}

// String satisfies the Stringer interface
func (di Diffs) String() string {
	var b strings.Builder
	for _, df := range di {
		switch df.Tag {
		case 'r':
			fmt.Fprintf(&b, "delete lines: %v - %v, insert lines: %v - %v\n", df.I1, df.I2, df.J1, df.J2)
		case 'd':
			fmt.Fprintf(&b, "delete lines: %v - %v\n", df.I1, df.I2)
		case 'i':
			fmt.Fprintf(&b, "insert lines at %v: %v - %v\n", df.I1, df.J1, df.J2)
		case 'e':
			fmt.Fprintf(&b, "same lines %v - %v == %v - %v\n", df.I1, df.I2, df.J1, df.J2)
		}
	}
	return b.String()
}

// DiffLines computes the diff between two string arrays (one string per line),
// reporting a sequence of operations that would convert buffer a into buffer b.
// Each operation is either an 'r' (replace), 'd' (delete), 'i' (insert)
// or 'e' (equal).  Everything is line-based (0, offset).
func DiffLines(astr, bstr []string) Diffs {
	m := difflib.NewMatcherWithJunk(astr, bstr, false, nil) // no junk
	return m.GetOpCodes()
}

// DiffLinesUnified computes the diff between two string arrays (one string per line),
// returning a unified diff with given amount of context (default of 3 will be
// used if -1), with given file names and modification dates.
func DiffLinesUnified(astr, bstr []string, context int, afile, adate, bfile, bdate string) []byte {
	ud := difflib.UnifiedDiff{A: astr, FromFile: afile, FromDate: adate,
		B: bstr, ToFile: bfile, ToDate: bdate, Context: context}
	var buf bytes.Buffer
	difflib.WriteUnifiedDiff(&buf, ud)
	return buf.Bytes()
}
