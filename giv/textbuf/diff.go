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

// Reverse returns the reverse-direction diffs, switching a vs. b
func (di Diffs) Reverse() Diffs {
	rd := make(Diffs, len(di))
	for i := range di {
		op := di[i]
		op.J1, op.I1 = op.I1, op.J1 // swap
		op.J2, op.I2 = op.I2, op.J2 // swap
		t := op.Tag
		switch t {
		case 'd':
			op.Tag = 'i'
		case 'i':
			op.Tag = 'd'
		}
		rd[i] = op
	}
	return rd
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

// PatchRec is a self-contained record of a DiffLines result that contains
// the source lines of the b buffer needed to patch a into b
type PatchRec struct {
	Op     difflib.OpCode `desc:"diff operation: 'r', 'd', 'i', 'e'"`
	Blines []string       `desc:"lines from B buffer needed for 'r' and 'i' operations"`
}

// Patch is a collection of patch records needed to turn original a buffer into b
type Patch []*PatchRec

// NumBlines returns the total number of Blines source code in the patch
func (pt Patch) NumBlines() int {
	nl := 0
	for _, pr := range pt {
		nl += len(pr.Blines)
	}
	return nl
}

// ToPatch creates a Patch list from given Diffs output from DiffLines and the
// b strings from which the needed lines of source are copied.
// ApplyPatch with this on the a strings will result in the b strings.
// The resulting Patch is independent of bstr slice.
func (dif Diffs) ToPatch(bstr []string) Patch {
	pt := make(Patch, len(dif))
	for pi, op := range dif {
		pr := &PatchRec{Op: op}
		if op.Tag == 'r' || op.Tag == 'i' {
			nl := (op.J2 - op.J1)
			pr.Blines = make([]string, nl)
			for i := 0; i < nl; i++ {
				pr.Blines[i] = bstr[op.J1+i]
			}
		}
		pt[pi] = pr
	}
	return pt
}

// Apply applies given Patch to given file as list of strings
// this does no checking except range checking so it won't crash
// so if input string is not appropriate for given Patch, results
// may be nonsensical.
func (pt Patch) Apply(astr []string) []string {
	np := len(pt)
	if np == 0 {
		return astr
	}
	sz := len(astr)
	lr := pt[np-1]
	bstr := make([]string, lr.Op.J2)
	for _, pr := range pt {
		switch pr.Op.Tag {
		case 'e':
			nl := (pr.Op.J2 - pr.Op.J1)
			for i := 0; i < nl; i++ {
				if pr.Op.I1+i < sz {
					bstr[pr.Op.J1+i] = astr[pr.Op.I1+i]
				}
			}
		case 'r', 'i':
			nl := (pr.Op.J2 - pr.Op.J1)
			for i := 0; i < nl; i++ {
				bstr[pr.Op.J1+i] = pr.Blines[i]
			}
		}
	}
	return bstr
}
