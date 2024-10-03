// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gotosl

import (
	"bytes"
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

// SlEdits performs post-generation edits for wgsl
// * moves wgsl segments around, e.g., methods
// into their proper classes
// * fixes printf, slice other common code
// returns true if a slrand. or sltype. prefix was found,
// driveing copying of those files.
func SlEdits(src []byte) ([]byte, bool, bool) {
	// return src // uncomment to show original without edits
	nl := []byte("\n")
	lines := bytes.Split(src, nl)
	hasSlrand, hasSltype := SlEditsReplace(lines)

	return bytes.Join(lines, nl), hasSlrand, hasSltype
}

type Replace struct {
	From, To []byte
}

var Replaces = []Replace{
	{[]byte("sltype.Uint32Vec2"), []byte("vec2<u32>")},
	{[]byte("sltype.Float32Vec2"), []byte("vec2<f32>")},
	{[]byte("float32"), []byte("f32")},
	{[]byte("float64"), []byte("f64")}, // TODO: not yet supported
	{[]byte("uint32"), []byte("u32")},
	{[]byte("uint64"), []byte("su64")},
	{[]byte("int32"), []byte("i32")},
	{[]byte("math32.FastExp("), []byte("FastExp(")}, // FastExp about same speed, numerically identical
	// {[]byte("math32.FastExp("), []byte("exp(")}, // exp is slightly faster it seems
	{[]byte("math.Float32frombits("), []byte("bitcast<f32>(")},
	{[]byte("math.Float32bits("), []byte("bitcast<u32>(")},
	{[]byte("shaders."), []byte("")},
	{[]byte("slrand."), []byte("Rand")},
	{[]byte("RandUi32"), []byte("RandUint32")}, // fix int32 -> i32
	{[]byte(".SetFromVector2("), []byte("=(")},
	{[]byte(".SetFrom2("), []byte("=(")},
	{[]byte(".IsTrue()"), []byte("==1")},
	{[]byte(".IsFalse()"), []byte("==0")},
	{[]byte(".SetBool(true)"), []byte("=1")},
	{[]byte(".SetBool(false)"), []byte("=0")},
	{[]byte(".SetBool("), []byte("=i32(")},
	{[]byte("slbool.Bool"), []byte("i32")},
	{[]byte("slbool.True"), []byte("1")},
	{[]byte("slbool.False"), []byte("0")},
	{[]byte("slbool.IsTrue("), []byte("(1 == ")},
	{[]byte("slbool.IsFalse("), []byte("(0 == ")},
	{[]byte("slbool.FromBool("), []byte("i32(")},
	{[]byte("bools.ToFloat32("), []byte("f32(")},
	{[]byte("bools.FromFloat32("), []byte("bool(")},
	{[]byte("num.FromBool[f32]("), []byte("f32(")},
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

// SlEditsReplace replaces Go with equivalent WGSL code
// returns true if has slrand. or sltype.
// to auto include that header file if so.
func SlEditsReplace(lines [][]byte) (bool, bool) {
	mt32 := []byte("math32.")
	mth := []byte("math.")
	slr := []byte("slrand.")
	styp := []byte("sltype.")
	include := []byte("#include")
	hasSlrand := false
	hasSltype := false
	for li, ln := range lines {
		if bytes.Contains(ln, include) {
			continue
		}
		for _, r := range Replaces {
			if !hasSlrand && bytes.Contains(ln, slr) {
				hasSlrand = true
			}
			if !hasSltype && bytes.Contains(ln, styp) {
				hasSltype = true
			}
			ln = bytes.ReplaceAll(ln, r.From, r.To)
		}
		ln = MathReplaceAll(mt32, ln)
		ln = MathReplaceAll(mth, ln)
		lines[li] = ln
	}
	return hasSlrand, hasSltype
}

var SLBools = []Replace{
	{[]byte(".IsTrue()"), []byte("==1")},
	{[]byte(".IsFalse()"), []byte("==0")},
	{[]byte(".SetBool(true)"), []byte("=1")},
	{[]byte(".SetBool(false)"), []byte("=0")},
	{[]byte(".SetBool("), []byte("=int32(")},
	{[]byte("slbool.Bool"), []byte("int32")},
	{[]byte("slbool.True"), []byte("1")},
	{[]byte("slbool.False"), []byte("0")},
	{[]byte("slbool.IsTrue("), []byte("(1 == ")},
	{[]byte("slbool.IsFalse("), []byte("(0 == ")},
	{[]byte("slbool.FromBool("), []byte("int32(")},
	{[]byte("bools.ToFloat32("), []byte("float32(")},
	{[]byte("bools.FromFloat32("), []byte("bool(")},
	{[]byte("num.FromBool[f32]("), []byte("float32(")},
	{[]byte("num.ToBool("), []byte("bool(")},
}

// SlBoolReplace replaces all the slbool methods with literal int32 expressions.
func SlBoolReplace(lines [][]byte) {
	for li, ln := range lines {
		for _, r := range SLBools {
			ln = bytes.ReplaceAll(ln, r.From, r.To)
		}
		lines[li] = ln
	}
}
