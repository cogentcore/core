// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gsbench

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type vec struct{ X, Y float32 }

func (v vec) Add(o vec) vec { return vec{v.X + o.X, v.Y + o.Y} }

type box struct {
	Min, Max vec
	Name     string
}

type shape interface{ Area() float32 }

func (b *box) Area() float32 { return (b.Max.X - b.Min.X) * (b.Max.Y - b.Min.Y) }

type circle struct{ R float32 }

func (c circle) Area() float32 { return 3.14159 * c.R * c.R }

var (
	sinkF   float32
	sinkI   int
	sinkS   string
	sinkB   box
	sinkAny any
)

// TestBench makes sure that this package has a test, so that it is not
// skipped as a package without tests.
func TestBench(t *testing.T) {
	if (vec{1, 2}).Add(vec{3, 4}) != (vec{4, 6}) {
		t.Fatal("bad add")
	}
}

func BenchmarkIntLoop(b *testing.B) {
	for range b.N {
		s := 0
		for i := range 1000 {
			s += i * i
		}
		sinkI = s
	}
}

func BenchmarkFloatLoop(b *testing.B) {
	for range b.N {
		var s float32
		for i := range 1000 {
			s += float32(i) * 0.5
		}
		sinkF = s
	}
}

func BenchmarkStructValueMethod(b *testing.B) {
	v := vec{1, 2}
	for range b.N {
		v = v.Add(vec{0.5, 0.25})
	}
	sinkF = v.X
}

func BenchmarkStructCopy(b *testing.B) {
	src := box{Min: vec{1, 2}, Max: vec{3, 4}, Name: "a"}
	for range b.N {
		dst := src
		dst.Min.X++
		sinkB = dst
	}
}

func BenchmarkStructSliceRange(b *testing.B) {
	boxes := make([]box, 100)
	for range b.N {
		var s float32
		for _, bx := range boxes {
			s += bx.Max.X
		}
		sinkF = s
	}
}

func BenchmarkInterfaceCall(b *testing.B) {
	shapes := []shape{&box{Max: vec{2, 3}}, circle{R: 2}}
	for range b.N {
		var s float32
		for _, sh := range shapes {
			s += sh.Area()
		}
		sinkF = s
	}
}

func BenchmarkTypeSwitch(b *testing.B) {
	vals := []any{1, "a", 2.5, &box{}, circle{}}
	for range b.N {
		n := 0
		for _, v := range vals {
			switch v.(type) {
			case int:
				n++
			case string:
				n += 2
			case *box:
				n += 3
			case shape:
				n += 4
			}
		}
		sinkI = n
	}
}

func BenchmarkTypeAssertInterface(b *testing.B) {
	vals := []any{&box{}, circle{}, 1}
	for range b.N {
		n := 0
		for _, v := range vals {
			if _, ok := v.(shape); ok {
				n++
			}
		}
		sinkI = n
	}
}

func BenchmarkMapStringGet(b *testing.B) {
	m := map[string]int{}
	keys := make([]string, 100)
	for i := range keys {
		keys[i] = "key" + strconv.Itoa(i)
		m[keys[i]] = i
	}
	for range b.N {
		s := 0
		for _, k := range keys {
			s += m[k]
		}
		sinkI = s
	}
}

func BenchmarkMapStructKey(b *testing.B) {
	m := map[vec]int{}
	for i := range 100 {
		m[vec{float32(i), 1}] = i
	}
	for range b.N {
		s := 0
		for i := range 100 {
			s += m[vec{float32(i), 1}]
		}
		sinkI = s
	}
}

func BenchmarkSliceAppend(b *testing.B) {
	for range b.N {
		var s []int
		for i := range 1000 {
			s = append(s, i)
		}
		sinkI = len(s)
	}
}

func BenchmarkPointerFieldUpdate(b *testing.B) {
	bx := &box{}
	for i := range b.N {
		bx.Min.X = float32(i)
		bx.Max.Y += bx.Min.X
	}
	sinkF = bx.Max.Y
}

func BenchmarkClosureCall(b *testing.B) {
	add := func(a, c int) int { return a + c }
	for range b.N {
		s := 0
		for i := range 100 {
			s = add(s, i)
		}
		sinkI = s
	}
}

func BenchmarkStringIndexLoop(b *testing.B) {
	s := strings.Repeat("hello world\n", 1000)
	for range b.N {
		n := 0
		for i := 0; i < len(s); i++ {
			if s[i] == '\n' {
				n++
			}
		}
		sinkI = n
	}
}

func BenchmarkStringRangeRunes(b *testing.B) {
	s := strings.Repeat("héllo wörld\n", 100)
	for range b.N {
		n := 0
		for _, r := range s {
			if r == 'ö' {
				n++
			}
		}
		sinkI = n
	}
}

func BenchmarkStringBuilder(b *testing.B) {
	for range b.N {
		var sb strings.Builder
		for range 100 {
			sb.WriteString("ab")
		}
		sinkS = sb.String()
	}
}

func BenchmarkStringConcat(b *testing.B) {
	for range b.N {
		s := ""
		for i := range 20 {
			s += strconv.Itoa(i)
		}
		sinkS = s
	}
}

func BenchmarkSprintf(b *testing.B) {
	for i := range b.N {
		sinkS = fmt.Sprintf("%s-%d-%g", "x", i, 1.5)
	}
}

func BenchmarkSortSlice(b *testing.B) {
	data := make([]int, 1000)
	for range b.N {
		for i := range data {
			data[i] = (i * 7919) % 1000
		}
		sort.Ints(data)
	}
}

func BenchmarkBoxInterface(b *testing.B) {
	for i := range b.N {
		sinkAny = vec{float32(i), 1}
	}
}

// BenchmarkMapPointerKeys fills and reads a map with interface keys that hold
// pointers to structs with slice and map fields, as tree.WalkDown does.
func BenchmarkMapPointerKeys(b *testing.B) {
	type node struct {
		Kids  []*node
		Attrs map[string]string
	}
	nodes := make([]*node, 200)
	for i := range nodes {
		nodes[i] = &node{}
	}
	for range b.N {
		m := map[any]int{}
		for i, n := range nodes {
			m[n] = i
		}
		s := 0
		for _, n := range nodes {
			s += m[n]
		}
		sinkI = s
	}
}
