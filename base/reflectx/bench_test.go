// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"reflect"
	"testing"
)

// These benchmarks measure the basic reflect operations that Cogent Core
// uses heavily, mainly to compare the native and goscript implementations.

type benchInner struct {
	A, B float32
}

type benchStruct struct {
	Name  string
	Value int
	Inner benchInner
	Items []string
	Ptr   *benchInner
}

func (b *benchStruct) Method() int { return b.Value }

var benchSink any

func BenchmarkValueOfPointer(b *testing.B) {
	v := &benchStruct{Name: "a"}
	for range b.N {
		benchSink = reflect.ValueOf(v)
	}
}

func BenchmarkTypeOfPointer(b *testing.B) {
	v := &benchStruct{Name: "a"}
	for range b.N {
		benchSink = reflect.TypeOf(v)
	}
}

func BenchmarkTypeOfStruct(b *testing.B) {
	v := benchStruct{Name: "a"}
	for range b.N {
		benchSink = reflect.TypeOf(v)
	}
}

func BenchmarkFieldByIndex(b *testing.B) {
	v := reflect.ValueOf(&benchStruct{Inner: benchInner{A: 1}}).Elem()
	for range b.N {
		benchSink = v.FieldByIndex([]int{2, 0}).Interface()
	}
}

func BenchmarkFieldByName(b *testing.B) {
	v := reflect.ValueOf(&benchStruct{Value: 3}).Elem()
	for range b.N {
		benchSink = v.FieldByName("Value").Int()
	}
}

func BenchmarkSetField(b *testing.B) {
	s := &benchStruct{}
	v := reflect.ValueOf(s).Elem().Field(1)
	for i := range b.N {
		v.SetInt(int64(i))
	}
}

func BenchmarkNumFieldLoop(b *testing.B) {
	t := reflect.TypeOf(benchStruct{})
	for range b.N {
		for i := range t.NumField() {
			benchSink = t.Field(i).Name
		}
	}
}

func BenchmarkMethodByName(b *testing.B) {
	v := reflect.ValueOf(&benchStruct{Value: 3})
	for range b.N {
		benchSink = v.MethodByName("Method")
	}
}

func BenchmarkValueInterface(b *testing.B) {
	v := reflect.ValueOf(&benchStruct{Value: 3}).Elem().Field(0)
	for range b.N {
		benchSink = v.Interface()
	}
}

func BenchmarkNonPointerType(b *testing.B) {
	t := reflect.TypeOf(&benchStruct{})
	for range b.N {
		benchSink = NonPointerType(t)
	}
}

func BenchmarkUnderlying(b *testing.B) {
	v := reflect.ValueOf(&benchStruct{})
	for range b.N {
		benchSink = Underlying(v)
	}
}

func BenchmarkCopyStruct(b *testing.B) {
	src := &benchStruct{Name: "a", Value: 2, Items: []string{"x"}}
	for range b.N {
		dst := &benchStruct{}
		reflect.ValueOf(dst).Elem().Set(reflect.ValueOf(src).Elem())
		benchSink = dst
	}
}
