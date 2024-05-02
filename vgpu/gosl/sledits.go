// Copyright 2022 The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"strings"
)

// MoveLines moves the st,ed region to 'to' line
func MoveLines(lines *[][]byte, to, st, ed int) {
	mvln := (*lines)[st:ed]
	btwn := (*lines)[to:st]
	aft := (*lines)[ed:len(*lines)]
	nln := make([][]byte, to, len(*lines))
	copy(nln, (*lines)[:to])
	nln = append(nln, mvln...)
	nln = append(nln, btwn...)
	nln = append(nln, aft...)
	*lines = nln
}

// SlEdits performs post-generation edits for hlsl
// * moves hlsl segments around, e.g., methods
// into their proper classes
// * fixes printf, slice other common code
// returns true if a slrand. prefix was found -- drives copying
// of that file.
func SlEdits(src []byte) ([]byte, bool) {
	// return src // uncomment to show original without edits
	nl := []byte("\n")
	lines := bytes.Split(src, nl)

	lines = SlEditsMethMove(lines)
	hasSlrand := SlEditsReplace(lines)

	return bytes.Join(lines, nl), hasSlrand
}

// SlEditsMethMove moves hlsl segments around, e.g., methods
// into their proper classes
func SlEditsMethMove(lines [][]byte) [][]byte {
	type sted struct {
		st, ed int
	}
	classes := map[string]sted{}

	class := []byte("struct ")
	slmark := []byte("<<<<")
	slend := []byte(">>>>")

	endclass := "EndClass: "
	method := "Method: "
	endmethod := "EndMethod"

	lastMethSt := -1
	var lastMeth string
	curComSt := -1
	lastComSt := -1
	lastComEd := -1

	li := 0
	for {
		if li >= len(lines) {
			break
		}
		ln := lines[li]
		if len(ln) >= 2 && string(ln[0:1]) == "//" {
			if curComSt >= 0 {
				lastComEd = li
			} else {
				curComSt = li
				lastComSt = li
				lastComEd = li
			}
		} else {
			curComSt = -1
		}

		switch {
		case bytes.HasPrefix(ln, class):
			cl := string(ln[len(class):])
			if idx := strings.Index(cl, "("); idx > 0 {
				cl = cl[:idx]
			} else if idx := strings.Index(cl, "{"); idx > 0 { // should have
				cl = cl[:idx]
			}
			cl = strings.TrimSpace(cl)
			classes[cl] = sted{st: li}
			// fmt.Printf("cl: %s at %d\n", cl, li)
		case bytes.HasPrefix(ln, slmark):
			sli := bytes.Index(ln, slend)
			if sli < 0 {
				continue
			}
			tag := string(ln[4:sli])
			// fmt.Printf("tag: %s at: %d\n", tag, li)
			switch {
			case strings.HasPrefix(tag, endclass):
				cl := tag[len(endclass):]
				st := classes[cl]
				classes[cl] = sted{st: st.st, ed: li - 1}
				lines = append(lines[:li], lines[li+1:]...) // delete marker
				// fmt.Printf("cl: %s at %v\n", cl, classes[cl])
				li--
			case strings.HasPrefix(tag, method):
				cl := tag[len(method):]
				lines = append(lines[:li], lines[li+1:]...) // delete marker
				li--
				lastMeth = cl
				if lastComEd == li {
					lines = append(lines[:lastComSt], lines[lastComEd+1:]...) // delete comments
					lastMethSt = lastComSt
					li = lastComSt - 1
				} else {
					lastMethSt = li + 1
				}
			case tag == endmethod:
				se, ok := classes[lastMeth]
				if ok {
					lines = append(lines[:li], lines[li+1:]...) // delete marker
					MoveLines(&lines, se.ed, lastMethSt, li+1)  // extra blank
					classes[lastMeth] = sted{st: se.st, ed: se.ed + ((li + 1) - lastMethSt)}
					li -= 2
				}
			}
		}
		li++
	}
	return lines
}

type Replace struct {
	From, To []byte
}

var Replaces = []Replace{
	{[]byte("float32"), []byte("float")},
	{[]byte("float64"), []byte("double")},
	{[]byte("uint32"), []byte("uint")},
	{[]byte("int32"), []byte("int")},
	{[]byte("int64"), []byte("int64_t")},
	{[]byte("math32.FastExp("), []byte("FastExp(")}, // FastExp about same speed, numerically identical
	// {[]byte("math32.FastExp("), []byte("exp(")}, // exp is slightly faster it seems
	{[]byte("math.Float32frombits("), []byte("asfloat(")},
	{[]byte("math.Float32bits("), []byte("asuint(")},
	{[]byte("shaders."), []byte("")},
	{[]byte("slrand."), []byte("Rand")},
	{[]byte("sltype.U"), []byte("u")},
	{[]byte("sltype.F"), []byte("f")},
	{[]byte(".SetFromVector2("), []byte("=(")},
	{[]byte(".SetFrom2("), []byte("=(")},
	{[]byte(".IsTrue()"), []byte("==1")},
	{[]byte(".IsFalse()"), []byte("==0")},
	{[]byte(".SetBool(true)"), []byte("=1")},
	{[]byte(".SetBool(false)"), []byte("=0")},
	{[]byte(".SetBool("), []byte("=int(")},
	{[]byte("slbool.Bool"), []byte("int")},
	{[]byte("slbool.True"), []byte("1")},
	{[]byte("slbool.False"), []byte("0")},
	{[]byte("slbool.IsTrue("), []byte("(1 == ")},
	{[]byte("slbool.IsFalse("), []byte("(0 == ")},
	{[]byte("slbool.FromBool("), []byte("int(")},
	{[]byte("bools.ToFloat32("), []byte("float(")},
	{[]byte("bools.FromFloat32("), []byte("bool(")},
	{[]byte("num.FromBool[float]("), []byte("float(")},
	{[]byte("num.ToBool("), []byte("bool(")},
	// todo: do this conversion in nodes only for correct types
	// {[]byte(".X"), []byte(".x")},
	// {[]byte(".Y"), []byte(".y")},
	// {[]byte(".Z"), []byte(".z")},
	// {[]byte(""), []byte("")},
	// {[]byte(""), []byte("")},
	// {[]byte(""), []byte("")},
}

func MathReplaceAll(mat, ln []byte) []byte {
	ml := len(mat)
	st := 0
	for {
		sln := ln[st:]
		i := bytes.Index(sln, mat)
		if i < 0 {
			return ln
		}
		fl := ln[st+i+ml : st+i+ml+1]
		dl := bytes.ToLower(fl)
		el := ln[st+i+ml+1:]
		ln = append(ln[:st+i], dl...)
		ln = append(ln, el...)
		st += i + 1
	}
}

// SlEditsReplace replaces Go with equivalent HLSL code
// returns true if has slrand. -- auto include that header file
// if so.
func SlEditsReplace(lines [][]byte) bool {
	mt32 := []byte("math32.")
	mth := []byte("math.")
	slr := []byte("slrand.")
	include := []byte("#include")
	hasSlrand := false
	for li, ln := range lines {
		if bytes.Contains(ln, include) {
			continue
		}
		for _, r := range Replaces {
			if !hasSlrand && bytes.Contains(ln, slr) {
				hasSlrand = true
			}
			ln = bytes.ReplaceAll(ln, r.From, r.To)
		}
		ln = MathReplaceAll(mt32, ln)
		ln = MathReplaceAll(mth, ln)
		lines[li] = ln
	}
	return hasSlrand
}
