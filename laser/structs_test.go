// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

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
	NmdA A
}

var a = A{}
var b = B{}
var c = C{}
var d = D{}

func InitC() {
	c.Mbr1 = "mbr1 string"
	c.Mbr2 = 2
	c.Mbr3 = "mbr3 string"
	c.Mbr4 = 4
	c.Mbr5 = "mbr5 string"
	c.Mbr6 = 6
}

func InitD() {
	d.Mbr5 = "mbr5 string"
	d.Mbr6 = 6
	d.NmdA.Mbr1 = "a in d"
	d.NmdA.Mbr2 = 2
}

func TestTypeEmbeds(t *testing.T) {
	InitC()

	a_in_a := TypeEmbeds(reflect.TypeOf(a), reflect.TypeOf(a))
	// fmt.Printf("A embeds A: %v\n", a_in_a)

	b_in_a := TypeEmbeds(reflect.TypeOf(a), reflect.TypeOf(b))
	// fmt.Printf("A embeds B: %v\n", b_in_a)

	a_in_b := TypeEmbeds(reflect.TypeOf(b), reflect.TypeOf(a))
	// fmt.Printf("B embeds A: %v\n", a_in_b)

	a_in_c := TypeEmbeds(reflect.TypeOf(c), reflect.TypeOf(a))
	// fmt.Printf("C embeds A: %v\n", a_in_c)

	aiftype := reflect.TypeOf((*AIf)(nil)).Elem()

	// note: MUST use pointer for checking implements for pointer receivers!
	// fmt.Printf("a implements Aif %v\n", reflect.TypeOf(&a).Implements(aiftype))

	aif_in_c := EmbedImplements(reflect.TypeOf(c), aiftype)
	// fmt.Printf("C implements AIf: %v\n", aif_in_c)

	aif_in_d := EmbedImplements(reflect.TypeOf(d), aiftype)
	// fmt.Printf("D implements AIf: %v\n", aif_in_d)

	if a_in_a != true || b_in_a != false || a_in_b != true || a_in_c != true || aif_in_c != true || aif_in_d != false {
		t.Errorf("something wrong in TypeEmbeds: should have: true, false, true, true, true, false is: %v %v %v %v %v %v\n", a_in_a, b_in_a, a_in_b, a_in_c, aif_in_c, aif_in_d)
	}
}

func TestEmbed(t *testing.T) {
	InitC()

	aa := Embed(&a, reflect.TypeOf(a))
	aas := fmt.Sprintf("%+v", aa)
	aat := "&{Mbr1: Mbr2:0}"
	if aas != aat {
		t.Errorf("Didn't get proper embedded members of A from A: %v != %v\n", aas, aat)
	}

	ca := Embed(&c, reflect.TypeOf(a))
	cas := fmt.Sprintf("%+v", ca)
	cat := "&{Mbr1:mbr1 string Mbr2:2}"
	if cas != cat {
		t.Errorf("Didn't get proper embedded members of C from A: %v != %v\n", cas, cat)
	}
}

func TestFlatFields(t *testing.T) {
	InitC()

	// FlatFieldsTypeFunc(reflect.TypeOf(c), func(typ reflect.Type, field reflect.StructField) {
	// 	fmt.Printf("typ: %v, field: %v\n", typ, field)
	// })

	// FlatFieldsValueFunc(c, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) {
	// 	fmt.Printf("typ: %v, field: %v val: %v\n", typ, field, fieldVal)
	// })

	// note: these test the above TypeFun and ValueFun

	ff := FlatFields(reflect.TypeOf(c))
	ffs := fmt.Sprintf("%v", ff)
	fft := `[{Mbr1  string  0 [0] false} {Mbr2  int  16 [1] false} {Mbr3  string  24 [1] false} {Mbr4  int  40 [2] false} {Mbr5  string  48 [1] false} {Mbr6  int  64 [2] false}]`
	if ffs != fft {
		t.Errorf("Didn't get proper flat field list of C: %v != %v\n", ffs, fft)
	}

	ffv := FlatFieldVals(&c)
	ffvs := fmt.Sprintf("%v", ffv)
	ffvt := `[mbr1 string <int Value> mbr3 string <int Value> mbr5 string <int Value>]`
	if ffvs != ffvt {
		t.Errorf("Didn't get proper flat field value list of C: %v != %v\n", ffvs, ffvt)
	}

	ffi := FlatFieldInterfaces(&c)
	ffis := ""
	for _, fi := range ffi {
		ffis += fmt.Sprintf("%v,", NonPtrInterface(fi))
	}
	ffit := `mbr1 string,2,mbr3 string,4,mbr5 string,6,`
	if ffis != ffit {
		t.Errorf("Didn't get proper flat field interface list of C: %v != %v\n", ffis, ffit)
	}
}

func TestFlatFieldsByName(t *testing.T) {
	InitC()

	fif, _ := FlatFieldByName(reflect.TypeOf(c), "Mbr3")
	fifs := fmt.Sprintf("%v", fif)
	fift := `{Mbr3  string  24 [0 1] false}`
	if fifs != fift {
		t.Errorf("Didn't get proper find flat field by name: %v != %v\n", fifs, fift)
	}

	fifn, _ := FlatFieldByName(reflect.TypeOf(c), "Mbr31")
	fifns := fmt.Sprintf("%v", fifn)
	fifnt := `{  <nil>  0 [] false}`
	if fifns != fifnt {
		t.Errorf("Didn't get proper nil find flat field by name: %v != %v\n", fifns, fifnt)
	}

	fifv := FlatFieldValueByName(&c, "Mbr4")
	fifvs := fmt.Sprintf("%v", fifv)
	fifvt := `4`
	if fifvs != fifvt {
		t.Errorf("Didn't get proper find flat field value by name: %v != %v\n", fifvs, fifvt)
	}

	fifi := FlatFieldInterfaceByName(&c, "Mbr2")
	fifis := fmt.Sprintf("%v", NonPtrInterface(fifi))
	fifit := `2`
	if fifis != fifit {
		t.Errorf("Didn't get proper find flat field value by name: %v != %v\n", fifis, fifit)
	}

}

func TestFieldPaths(t *testing.T) {
	InitD()

	fld, ok := FieldByPath(reflect.TypeOf(d), "NmdA.Mbr1")
	if !ok {
		t.Errorf("FieldByPath failed per err msg, fld %v\n", fld.Name)
	}

	fi, ok := FieldValueByPath(d, "NmdA.Mbr1")
	if !ok {
		t.Errorf("FieldValueByPath failed per err msg, fi %v\n", fi)
	}
	// fmt.Printf("fi: %v\n", fi)
}

type person struct {
	Name                string `default:"Go Gopher"`
	Age                 int    `default:"35"`
	ProgrammingLanguage string `default:"Go"`
	Pet                 pet
	FavoriteFruit       string `default:"Apple"`
	Data                string `save:"-"`
	OtherPet            *pet
}

type pet struct {
	Name       string
	Type       string `default:"Gopher"`
	Age        int    `default:"7"`
	IsSick     bool
	LikesFoods []string
}

func TestNonDefaultFields(t *testing.T) {
	p := &person{
		Name:                "Go Gopher",
		Age:                 23,
		ProgrammingLanguage: "Go",
		FavoriteFruit:       "Peach",
		Data:                "abcdef",
		Pet: pet{
			Name: "Pet Gopher",
			Type: "Dog",
			Age:  7,
		},
	}
	want := map[string]any{
		"Age":           23,
		"FavoriteFruit": "Peach",
		"Pet": map[string]any{
			"Name": "Pet Gopher",
			"Type": "Dog",
		},
	}
	have := NonDefaultFields(p)
	if !reflect.DeepEqual(have, want) {
		t.Errorf("expected\n%v\n\tbut got\n%v", want, have)
	}
}
