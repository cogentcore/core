// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"fmt"
	"reflect"
	"testing"
)

type A struct {
	Mbr1 string
	Mbr2 int
}

type AIf interface {
	AFun() bool
}

func (a *A) AFun() bool {
	return true
}

var _ AIf = &A{}

type B struct {
	A
	Mbr3 string
	Mbr4 int
}

type C struct {
	B
	Mbr5 string
	Mbr6 int
}

type D struct {
	Mbr5 string
	Mbr6 int
}

func TestTypeEmbeds(t *testing.T) {
	a := A{}
	b := B{}
	c := C{}
	d := D{}

	c.Mbr1 = "a string"
	c.Mbr2 = 42

	b_in_a := TypeEmbeds(reflect.TypeOf(a), reflect.TypeOf(b))
	// fmt.Printf("A embeds B: %v\n", b_in_a)

	a_in_b := TypeEmbeds(reflect.TypeOf(b), reflect.TypeOf(a))
	// fmt.Printf("B embeds A: %v\n", a_in_b)

	a_in_c := TypeEmbeds(reflect.TypeOf(c), reflect.TypeOf(a))
	// fmt.Printf("C embeds A: %v\n", a_in_c)

	aiftype := reflect.TypeOf((*AIf)(nil)).Elem()

	// note: MUST use pointer for checking implements for pointer receivers!
	// fmt.Printf("a implements Aif %v\n", reflect.TypeOf(&a).Implements(aiftype))

	aif_in_c := EmbeddedTypeImplements(reflect.TypeOf(c), aiftype)
	// fmt.Printf("C implements AIf: %v\n", aif_in_c)

	aif_in_d := EmbeddedTypeImplements(reflect.TypeOf(d), aiftype)
	// fmt.Printf("D implements AIf: %v\n", aif_in_d)

	if b_in_a != false || a_in_b != true || a_in_c != true || aif_in_c != true || aif_in_d != false {
		t.Errorf("something wrong in TypeEmbeds: should have false, true, true, true, false is: %v %v %v %v %v\n", b_in_a, a_in_b, a_in_c, aif_in_c, aif_in_d)
	}

	ca := EmbededStruct(&c, reflect.TypeOf(a))
	cas := fmt.Sprintf("%+v", ca)

	cat := "{Mbr1:a string Mbr2:42}"

	if cas != cat {
		t.Errorf("Didn't get proper embedded members of C from At: %v != %v\n", cas, cat)
	}

	// FlatFieldsTypeFun(reflect.TypeOf(c), func(typ reflect.Type, field reflect.StructField) {
	// 	fmt.Printf("typ: %v, field: %v\n", typ, field)
	// })

	// FlatFieldsValueFun(c, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) {
	// 	fmt.Printf("typ: %v, field: %v val: %v\n", typ, field, fieldVal)
	// })
}
