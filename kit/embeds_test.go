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

func TestTypeEmbeds(t *testing.T) {
	a := A{}
	b := B{}
	c := C{}

	c.Mbr1 = "a string"
	c.Mbr2 = 42

	b_in_a := TypeEmbeds(reflect.TypeOf(a), reflect.TypeOf(b))
	// fmt.Printf("A embeds B: %v\n", b_in_a)

	a_in_b := TypeEmbeds(reflect.TypeOf(b), reflect.TypeOf(a))
	// fmt.Printf("B embeds A: %v\n", a_in_b)

	a_in_c := TypeEmbeds(reflect.TypeOf(c), reflect.TypeOf(a))
	// fmt.Printf("C embeds A: %v\n", a_in_c)

	if b_in_a != false || a_in_b != true || a_in_c != true {
		t.Errorf("something wrong in TypeEmbeds: should have false, true, true, is: %v %v %v\n", b_in_a, a_in_b, a_in_c)
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
