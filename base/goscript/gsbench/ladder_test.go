// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gsbench

import (
	"fmt"
	"testing"
)

// The Ladder benchmarks start from the Baseline benchmarks and add one
// language feature at a time, up to the kind of code that the text/parse
// lexer runs, to find the features that make code compiled to JavaScript
// much slower than WebAssembly. Each one does about 10000 basic steps.

var (
	ladSinkI int
	ladSinkB bool
	ladSinkS string
)

// ladPos is a small struct of scalar fields, like a lexer position.
type ladPos struct {
	Line, Char int
}

// ladState is a lexer-like state: a line of runes and a position in it.
type ladState struct {
	Src  []rune
	Pos  int
	Rune rune
	Name string
	Last ladPos
}

// ladSrc returns a line of source for the string benchmarks.
func ladSrc() string {
	return "func (ls *State) ReadName() { st := ls.Pos; for ls.Pos < sz { ls.Pos++ } } // 1234"
}

// LadderStructValue reads and writes the fields of a local struct value.
func BenchmarkLadderStructValue(b *testing.B) {
	for range b.N {
		var p ladPos
		for i := range 10000 {
			p.Line += i & 7
			p.Char = p.Line ^ i
		}
		ladSinkI = p.Line + p.Char
	}
}

// LadderStructPointer reads and writes the fields of a struct through a
// pointer to a new struct.
func BenchmarkLadderStructPointer(b *testing.B) {
	for range b.N {
		p := &ladPos{}
		for i := range 10000 {
			p.Line += i & 7
			p.Char = p.Line ^ i
		}
		ladSinkI = p.Line + p.Char
	}
}

// LadderVarPointer reads and writes the fields of a struct variable through
// a pointer to the variable.
func BenchmarkLadderVarPointer(b *testing.B) {
	for range b.N {
		var v ladPos
		p := &v
		for i := range 10000 {
			p.Line += i & 7
			p.Char = p.Line ^ i
		}
		ladSinkI = v.Line + v.Char
	}
}

// ladOuter has a struct field, as parse.FileState has the lexer State.
type ladOuter struct {
	Name string
	St   ladState
}

// LadderFieldPointer reads and writes the fields of a struct through a
// pointer to a struct field of another struct (as &fs.LexState).
func BenchmarkLadderFieldPointer(b *testing.B) {
	o := &ladOuter{}
	for range b.N {
		p := &o.St
		p.Pos = 0
		for i := range 10000 {
			p.Pos += i & 7
			p.Rune = rune(p.Pos ^ i)
		}
		ladSinkI = p.Pos + int(p.Rune)
	}
}

// ladStep is a value receiver method.
//
//go:noinline
func (p ladPos) ladStep(i int) int { return p.Line + i&7 }

// ladAdvance is a pointer receiver method.
//
//go:noinline
func (p *ladPos) ladAdvance(i int) { p.Line += i & 7 }

// LadderValueMethod calls a method with a struct value receiver.
func BenchmarkLadderValueMethod(b *testing.B) {
	for range b.N {
		p := ladPos{Line: 1}
		s := 0
		for i := range 10000 {
			s += p.ladStep(i)
		}
		ladSinkI = s
	}
}

// LadderPointerMethod calls a method with a pointer receiver.
func BenchmarkLadderPointerMethod(b *testing.B) {
	for range b.N {
		p := &ladPos{}
		for i := range 10000 {
			p.ladAdvance(i)
		}
		ladSinkI = p.Line
	}
}

// ladStepper is an interface with one method.
type ladStepper interface{ ladAdvance(i int) }

// LadderInterfaceMethod calls a pointer method through an interface.
func BenchmarkLadderInterfaceMethod(b *testing.B) {
	for range b.N {
		p := &ladPos{}
		var st ladStepper = p
		for i := range 10000 {
			st.ladAdvance(i)
		}
		ladSinkI = p.Line
	}
}

// LadderStructSliceIndex reads and writes the fields of struct elements of
// a slice by index.
func BenchmarkLadderStructSliceIndex(b *testing.B) {
	ps := make([]ladPos, 100)
	for range b.N {
		for i := range 10000 {
			j := i % 100
			ps[j].Line += i & 7
			ps[j].Char = ps[j].Line
		}
		ladSinkI = ps[7].Char
	}
}

// LadderStructSliceAddr takes the address of struct elements of a slice
// (&s[i]) and uses the fields through it.
func BenchmarkLadderStructSliceAddr(b *testing.B) {
	ps := make([]ladPos, 100)
	for range b.N {
		for i := range 10000 {
			p := &ps[i%100]
			p.Line += i & 7
			p.Char = p.Line
		}
		ladSinkI = ps[7].Char
	}
}

// LadderStructAppend appends struct values to a slice, as the lexer appends
// tokens to a line.
func BenchmarkLadderStructAppend(b *testing.B) {
	for range b.N {
		var ps []ladPos
		for i := range 10000 {
			ps = append(ps, ladPos{Line: i, Char: i & 7})
		}
		ladSinkI = len(ps)
	}
}

// LadderStringIndex reads the bytes of a string by index.
func BenchmarkLadderStringIndex(b *testing.B) {
	src := ladSrc()
	for range b.N {
		s := 0
		for i := range 10000 {
			s += int(src[i%len(src)])
		}
		ladSinkI = s
	}
}

// LadderStringRange ranges over the runes of short strings, as the lexer
// matches literal strings.
func BenchmarkLadderStringRange(b *testing.B) {
	words := []string{"func", "if", "for", "range", ":=", "//"}
	for range b.N {
		s := 0
		for i := range 2000 {
			for _, r := range words[i%len(words)] {
				s += int(r)
			}
		}
		ladSinkI = s
	}
}

// LadderStringSliceCompare slices a string and compares the parts to a
// string.
func BenchmarkLadderStringSliceCompare(b *testing.B) {
	src := ladSrc()
	for range b.N {
		n := 0
		for i := range 10000 {
			j := i % (len(src) - 4)
			if src[j:j+4] == "func" {
				n++
			}
		}
		ladSinkI = n
	}
}

// LadderRuneConvert converts a string to a []rune.
func BenchmarkLadderRuneConvert(b *testing.B) {
	src := ladSrc()
	for range b.N {
		n := 0
		for range 100 {
			n += len([]rune(src))
		}
		ladSinkI = n
	}
}

// LadderRuneIndex reads the elements of a []rune by index, as the lexer
// reads its line.
func BenchmarkLadderRuneIndex(b *testing.B) {
	src := []rune(ladSrc())
	for range b.N {
		s := 0
		for i := range 10000 {
			s += int(src[i%len(src)])
		}
		ladSinkI = s
	}
}

// LadderRuneFieldIndex reads the []rune line of a lexer-like state through
// a pointer, by its position, as State.RuneAt does.
func BenchmarkLadderRuneFieldIndex(b *testing.B) {
	st := &ladState{Src: []rune(ladSrc())}
	for range b.N {
		s := 0
		for i := range 10000 {
			st.Pos = i % len(st.Src)
			if st.Pos < len(st.Src) {
				s += int(st.Src[st.Pos])
			}
		}
		ladSinkI = s
	}
}

// LadderRunesToString makes strings from parts of a []rune, as the lexer
// reads names.
func BenchmarkLadderRunesToString(b *testing.B) {
	src := []rune(ladSrc())
	for range b.N {
		n := 0
		for i := range 1000 {
			j := i % (len(src) - 8)
			n += len(string(src[j : j+8]))
		}
		ladSinkI = n
	}
}

// LadderMapStringKey looks up string keys in a map.
func BenchmarkLadderMapStringKey(b *testing.B) {
	m := map[string]int{"func": 1, "if": 2, "for": 3, "range": 4, "return": 5, "var": 6}
	keys := []string{"func", "x", "range", "return", "foo", "var"}
	for range b.N {
		s := 0
		for i := range 10000 {
			if v, ok := m[keys[i%len(keys)]]; ok {
				s += v
			}
		}
		ladSinkI = s
	}
}

// ladRuneAt returns two values, as State.RuneAt does.
//
//go:noinline
func ladRuneAt(src []rune, i int) (rune, bool) {
	if i >= len(src) {
		return 0, false
	}
	return src[i], true
}

// LadderMultiReturn calls a function with two results.
func BenchmarkLadderMultiReturn(b *testing.B) {
	src := []rune(ladSrc())
	for range b.N {
		s := 0
		for i := range 10000 {
			if r, ok := ladRuneAt(src, i%100); ok {
				s += int(r)
			}
		}
		ladSinkI = s
	}
}

// ladMakePos returns a small struct.
//
//go:noinline
func ladMakePos(i int) ladPos { return ladPos{Line: i >> 4, Char: i & 15} }

// LadderStructReturn calls a function that returns a small struct.
func BenchmarkLadderStructReturn(b *testing.B) {
	for range b.N {
		s := 0
		for i := range 10000 {
			p := ladMakePos(i)
			s += p.Line + p.Char
		}
		ladSinkI = s
	}
}

// LadderClosure calls a closure that captures and changes a local variable.
func BenchmarkLadderClosure(b *testing.B) {
	for range b.N {
		s := 0
		add := func(i int) { s += i & 7 }
		for i := range 10000 {
			add(i)
		}
		ladSinkI = s
	}
}

// LadderNestedField reads and writes a field of a struct field of a struct
// through a pointer.
func BenchmarkLadderNestedField(b *testing.B) {
	o := &ladOuter{}
	for range b.N {
		for i := range 10000 {
			o.St.Last.Char += i & 7
		}
		ladSinkI = o.St.Last.Char
	}
}

// ladBase is embedded in ladEmbed.
type ladBase struct {
	N int
}

//go:noinline
func (b *ladBase) ladInc(i int) { b.N += i & 7 }

// ladEmbed embeds ladBase, as lexer.Rule embeds tree.NodeBase.
type ladEmbed struct {
	ladBase
	Name string
}

// LadderEmbedded uses a promoted field and calls a promoted method.
func BenchmarkLadderEmbedded(b *testing.B) {
	for range b.N {
		e := &ladEmbed{}
		for i := range 10000 {
			e.ladInc(i)
			e.N ^= 1
		}
		ladSinkI = e.N
	}
}

// ladNode is a tree node interface, as tree.Node.
type ladNode interface{ ladKids() []ladNode }

// ladRule is a lexer-like rule tree node.
type ladRule struct {
	Kids []ladNode
	Ch   rune
}

func (r *ladRule) ladKids() []ladNode { return r.Kids }

// ladRules returns a rule with 20 children.
func ladRules() *ladRule {
	root := &ladRule{}
	for i := range 20 {
		root.Kids = append(root.Kids, &ladRule{Ch: rune('a' + i)})
	}
	return root
}

// LadderTypeAssert ranges over interface children and asserts them to their
// pointer type, as lexer.Rule.Lex does with its children.
func BenchmarkLadderTypeAssert(b *testing.B) {
	root := ladRules()
	for range b.N {
		s := 0
		for range 500 {
			for _, k := range root.Kids {
				kr := k.(*ladRule)
				s += int(kr.Ch)
			}
		}
		ladSinkI = s
	}
}

// ladMatch is a lexer-like matcher: it calls fmt only on an error that
// never happens, which in JavaScript can make it an async function.
func (r *ladRule) ladMatch(st *ladState) bool {
	if st.Pos > len(st.Src) {
		ladSinkS = fmt.Sprintf("bad pos %d", st.Pos)
		return false
	}
	return st.Pos < len(st.Src) && st.Src[st.Pos] == r.Ch
}

// ladLex is a lexer-like recursive matcher over the rule tree.
func (r *ladRule) ladLex(st *ladState) *ladRule {
	for _, k := range r.Kids {
		kr := k.(*ladRule)
		if kr.ladMatch(st) {
			return kr
		}
	}
	return nil
}

// LadderLexLike matches each rune of a line of source against a tree of
// rules, as the lexer does.
func BenchmarkLadderLexLike(b *testing.B) {
	root := ladRules()
	st := &ladState{Src: []rune(ladSrc())}
	for range b.N {
		n := 0
		for range 25 {
			for st.Pos = 0; st.Pos < len(st.Src); st.Pos++ {
				if root.ladLex(st) != nil {
					n++
				}
			}
		}
		ladSinkI = n
	}
}

// ladMatchSync is ladMatch without the fmt call.
func (r *ladRule) ladMatchSync(st *ladState) bool {
	if st.Pos > len(st.Src) {
		ladSinkS = "bad pos"
		return false
	}
	return st.Pos < len(st.Src) && st.Src[st.Pos] == r.Ch
}

// ladLexSync is ladLex with ladMatchSync.
func (r *ladRule) ladLexSync(st *ladState) *ladRule {
	for _, k := range r.Kids {
		kr := k.(*ladRule)
		if kr.ladMatchSync(st) {
			return kr
		}
	}
	return nil
}

// LadderLexLikeSync is LadderLexLike without the fmt call, so that the
// matcher is not async in JavaScript.
func BenchmarkLadderLexLikeSync(b *testing.B) {
	root := ladRules()
	st := &ladState{Src: []rune(ladSrc())}
	for range b.N {
		n := 0
		for range 25 {
			for st.Pos = 0; st.Pos < len(st.Src); st.Pos++ {
				if root.ladLexSync(st) != nil {
					n++
				}
			}
		}
		ladSinkI = n
	}
}
