// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"image"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToString(t *testing.T) {
	tests := []struct {
		v    any
		want string
	}{
		{"s", "s"},
		{new("s"), "s"},
		{[]byte("b"), "b"},
		{new([]byte("b")), "b"},
		{time.Duration(0), "0s"}, // a Stringer
		{int(3), "3"}, {int8(-3), "-3"}, {int16(3), "3"}, {int32(3), "3"}, {int64(3), "3"},
		{uint(3), "3"}, {uint8('A'), "A"}, {uint16(3), "3"}, {uint32(3), "3"}, {uint64(3), "3"}, {uintptr(3), "0x3"},
		{new(int(3)), "3"}, {new(int8(3)), "3"}, {new(int16(3)), "3"}, {new(int32(3)), "3"}, {new(int64(3)), "3"},
		{new(uint(3)), "3"}, {new(uint8('A')), "A"}, {new(uint16(3)), "3"}, {new(uint32(3)), "3"}, {new(uint64(3)), "3"}, {new(uintptr(3)), "0x3"},
		{true, "true"}, {new(false), "false"},
		{1.5, "1.5"}, {float32(1.5), "1.5"}, {new(1.5), "1.5"}, {new(float32(1.5)), "1.5"},
		{complex(1, 2), "1,2"}, {complex64(complex(1, 2)), "1,2"},
		{new(complex(1, 2)), "1,2"}, {new(complex64(complex(1, 2))), "1,2"},
		{myInt(4), "4"}, {myString("m"), "m"},
		{[]int{1, 2}, "[1 2]"},
		{(*int)(nil), "nil"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ToString(tt.v), "%T", tt.v)
	}
}

func TestToStringPrec(t *testing.T) {
	tests := []struct {
		v    any
		want string
	}{
		{1.0 / 3, "0.333"},
		{float32(1.0 / 3), "0.333"},
		{new(1.0 / 3), "0.333"},
		{new(float32(1.0 / 3)), "0.333"},
		{complex(1.0/3, 2), "0.333,2"},
		{complex64(complex(1.0/3, 2)), "0.333,2"},
		{new(complex(1.0/3, 2)), "0.333,2"},
		{new(complex64(complex(1.0/3, 2))), "0.333,2"},
		{"s", "s"}, {new("s"), "s"}, {[]byte("b"), "b"}, {new([]byte("b")), "b"},
		{time.Second, "1s"},
		{7, "7"},
		{(*complex64)(nil), "nil"},
		{(*complex128)(nil), "nil"},
		{(*[]byte)(nil), "nil"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ToStringPrec(tt.v, 3), "%T", tt.v)
	}
}

func TestKinds(t *testing.T) {
	tests := []struct {
		k                     reflect.Kind
		basic, num, int, flot bool
	}{
		{reflect.Bool, true, false, false, false},
		{reflect.Int, true, true, true, false},
		{reflect.Uintptr, true, true, true, false},
		{reflect.Float32, true, true, false, true},
		{reflect.Complex128, true, true, false, false},
		{reflect.String, false, false, false, false},
		{reflect.Struct, false, false, false, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.basic, KindIsBasic(tt.k), "%v", tt.k)
		assert.Equal(t, tt.num, KindIsNumber(tt.k), "%v", tt.k)
		assert.Equal(t, tt.int, KindIsInt(tt.k), "%v", tt.k)
		assert.Equal(t, tt.flot, KindIsFloat(tt.k), "%v", tt.k)
	}
}

func TestTypeNames(t *testing.T) {
	pt := reflect.TypeFor[*image.Point]()
	assert.Equal(t, "image.Point", LongTypeName(pt))
	assert.Equal(t, "image.Point", ShortTypeName(pt))
	tt := reflect.TypeFor[time.Duration]()
	assert.Equal(t, "time.Duration", LongTypeName(tt))
	vt := reflect.TypeFor[myInt]()
	assert.Equal(t, "cogentcore.org/core/base/reflectx.myInt", LongTypeName(vt))
	assert.Equal(t, "reflectx.myInt", ShortTypeName(vt))
	assert.Equal(t, "int", LongTypeName(reflect.TypeFor[int]()))
	assert.Equal(t, "int", ShortTypeName(reflect.TypeFor[int]()))
	assert.Equal(t, "[]int", LongTypeName(reflect.TypeFor[[]int]()))
	assert.Equal(t, "[]int", ShortTypeName(reflect.TypeFor[[]int]()))

	cl := CloneToType(reflect.TypeFor[int](), "5")
	assert.Equal(t, 5, cl.Elem().Interface())
}

func TestMapAddDeleteSort(t *testing.T) {
	var m map[string]int
	MapAdd(&m)
	require.NotNil(t, m, "a nil map is made")
	assert.Equal(t, map[string]int{"": 0}, m)

	ma := map[string]any{}
	MapAdd(&ma)
	assert.Equal(t, "", ma[""], "an any value is a string")

	m = map[string]int{"b": 1, "a": 3, "c": 2}
	MapDelete(&m, reflect.ValueOf("b"))
	assert.Equal(t, map[string]int{"a": 3, "c": 2}, m)

	m = map[string]int{"b": 1, "a": 3, "c": 2}
	keys := func(vs []reflect.Value) []string {
		var s []string
		for _, v := range vs {
			s = append(s, v.String())
		}
		return s
	}
	assert.Equal(t, []string{"a", "b", "c"}, keys(MapSort(m, true, true)))
	assert.Equal(t, []string{"c", "b", "a"}, keys(MapSort(&m, true, false)))
	assert.Equal(t, []string{"b", "c", "a"}, keys(MapSort(m, false, true)))
	assert.Equal(t, []string{"a", "c", "b"}, keys(MapSort(m, false, false)))

	mi := map[int]float64{3: 0.5, 1: 2.5, 2: 1.5}
	ki := func(vs []reflect.Value) []int64 {
		var s []int64
		for _, v := range vs {
			s = append(s, v.Int())
		}
		return s
	}
	assert.Equal(t, []int64{1, 2, 3}, ki(MapSort(mi, true, true)))
	assert.Equal(t, []int64{3, 2, 1}, ki(MapSort(mi, false, true)))
	assert.Equal(t, []int64{1, 2, 3}, ki(MapSort(mi, false, false)))

	ms := map[string]string{"x": "b", "y": "a"}
	assert.Equal(t, []string{"y", "x"}, keys(MapSort(ms, false, true)))

	mu := map[uint]uint{2: 1, 1: 2}
	assert.Len(t, MapSort(mu, true, true), 2)
	assert.Len(t, MapSort(mu, false, true), 2)
	mt := map[string]time.Time{"late": time.Unix(200, 0), "early": time.Unix(100, 0)}
	assert.Equal(t, []string{"early", "late"}, keys(MapSort(mt, false, true)))
	assert.Equal(t, []string{"late", "early"}, keys(MapSort(mt, false, false)))
	assert.NoError(t, MapValueSort(reflect.ValueOf(m), nil, true))
}

func TestSetMapRobust(t *testing.T) {
	m := map[int]float64{}
	mv := reflect.ValueOf(&m).Elem()
	assert.True(t, SetMapRobust(mv, reflect.ValueOf(1), reflect.ValueOf(1.5)))
	assert.True(t, SetMapRobust(mv, reflect.ValueOf(2), reflect.ValueOf(int32(2))), "the value is converted")
	assert.True(t, SetMapRobust(mv, reflect.ValueOf(3.0), reflect.ValueOf(3.0)), "the key is converted")
	assert.True(t, SetMapRobust(mv, reflect.ValueOf(4.0), reflect.ValueOf(4)), "both are converted")
	assert.Equal(t, map[int]float64{1: 1.5, 2: 2, 3: 3, 4: 4}, m)

	assert.False(t, SetMapRobust(reflect.ValueOf(1), reflect.ValueOf("a"), reflect.ValueOf(1)), "not a map")
	assert.False(t, SetMapRobust(reflect.ValueOf(m), reflect.ValueOf(1), reflect.ValueOf(1.0)), "not settable")
}

// TestSetMapRobustNamedKind tests setting a map with a key or value of a
// named type of the same kind as that of the map.
func TestSetMapRobustNamedKind(t *testing.T) {
	m := map[string]float64{}
	mv := reflect.ValueOf(&m).Elem()
	assert.True(t, SetMapRobust(mv, reflect.ValueOf(myString("c")), reflect.ValueOf(3.0)))
	assert.Equal(t, 3.0, m["c"])
}

func TestSliceDeleteSort(t *testing.T) {
	s := []int{1, 2, 3, 4}
	SliceDeleteAt(&s, 1)
	assert.Equal(t, []int{1, 3, 4}, s)
	SliceDeleteAt(&s, 2)
	assert.Equal(t, []int{1, 3}, s)

	si := []int{3, 1, 2}
	assert.NoError(t, SliceSort(si, true))
	assert.Equal(t, []int{1, 2, 3}, si)
	assert.NoError(t, SliceSort(&si, false))
	assert.Equal(t, []int{3, 2, 1}, si)

	su := []uint8{3, 1, 2}
	assert.NoError(t, SliceSort(su, true))
	assert.Equal(t, []uint8{1, 2, 3}, su)
	assert.NoError(t, SliceSort(su, false))
	assert.Equal(t, []uint8{3, 2, 1}, su)

	sf := []float32{0.5, 2, 1}
	assert.NoError(t, SliceSort(sf, true))
	assert.Equal(t, []float32{0.5, 1, 2}, sf)
	assert.NoError(t, SliceSort(sf, false))
	assert.Equal(t, []float32{2, 1, 0.5}, sf)

	ss := []string{"b", "c", "a"}
	assert.NoError(t, SliceSort(ss, true))
	assert.Equal(t, []string{"a", "b", "c"}, ss)
	assert.NoError(t, SliceSort(ss, false))
	assert.Equal(t, []string{"c", "b", "a"}, ss)

	st := []time.Time{time.Unix(3, 0), time.Unix(1, 0), time.Unix(2, 0)}
	assert.NoError(t, SliceSort(st, true))
	assert.Equal(t, time.Unix(1, 0), st[0])
	assert.NoError(t, SliceSort(st, false))
	assert.Equal(t, time.Unix(3, 0), st[0])

	assert.NoError(t, SliceSort([]int{}, true))
	assert.NoError(t, SliceSort([]image.Point{{1, 2}}, true), "a Stringer sorts by its string")
	assert.Error(t, SliceSort([]struct{ X int }{{1}}, true), "not a basic type")
}

func TestValueSliceSort(t *testing.T) {
	vals := func(vs ...any) []reflect.Value {
		var r []reflect.Value
		for _, v := range vs {
			r = append(r, reflect.ValueOf(v))
		}
		return r
	}
	first := func(vs []reflect.Value) any { return vs[0].Interface() }

	vi := vals(3, 1, 2)
	assert.NoError(t, ValueSliceSort(vi, true))
	assert.Equal(t, 1, first(vi))
	assert.NoError(t, ValueSliceSort(vi, false))
	assert.Equal(t, 3, first(vi))

	vu := vals(uint(3), uint(1))
	assert.NoError(t, ValueSliceSort(vu, true))
	assert.Equal(t, uint(1), first(vu))
	assert.NoError(t, ValueSliceSort(vu, false))
	assert.Equal(t, uint(3), first(vu))

	vf := vals(0.5, 2.0)
	assert.NoError(t, ValueSliceSort(vf, false))
	assert.Equal(t, 2.0, first(vf))
	assert.NoError(t, ValueSliceSort(vf, true))
	assert.Equal(t, 0.5, first(vf))

	vs := vals("b", "a")
	assert.NoError(t, ValueSliceSort(vs, true))
	assert.Equal(t, "a", first(vs))
	assert.NoError(t, ValueSliceSort(vs, false))
	assert.Equal(t, "b", first(vs))

	vt := vals(time.Unix(2, 0), time.Unix(1, 0))
	assert.NoError(t, ValueSliceSort(vt, true))
	assert.Equal(t, time.Unix(1, 0), first(vt))
	assert.NoError(t, ValueSliceSort(vt, false))
	assert.Equal(t, time.Unix(2, 0), first(vt))

	assert.NoError(t, ValueSliceSort(nil, true))
	assert.NoError(t, ValueSliceSort(vals(image.Point{}), true), "a Stringer")
	assert.Error(t, ValueSliceSort(vals(struct{ X int }{}), true))
}

// sortRow is a struct for testing StructSliceSort on each kind of field.
type sortRow struct {
	B bool
	I int
	U uint
	F float64
	S string
	T time.Time
	P image.Point
	N struct{ X int }
}

func TestStructSliceSortKinds(t *testing.T) {
	rows := []sortRow{
		{B: true, I: 2, U: 2, F: 2, S: "b", T: time.Unix(2, 0)},
		{B: false, I: 1, U: 3, F: 3, S: "c", T: time.Unix(3, 0)},
		{B: true, I: 3, U: 1, F: 1, S: "a", T: time.Unix(1, 0)},
	}
	firstI := func() int { return rows[0].I }
	for _, tt := range []struct {
		field     int
		asc, desc int // the I of the first row
	}{
		{1, 1, 3}, // I
		{2, 3, 1}, // U
		{3, 3, 1}, // F
		{4, 3, 1}, // S
		{5, 3, 1}, // T
	} {
		assert.NoError(t, StructSliceSort(rows, []int{tt.field}, true))
		assert.Equal(t, tt.asc, firstI(), "field %d ascending", tt.field)
		assert.NoError(t, StructSliceSort(&rows, []int{tt.field}, false))
		assert.Equal(t, tt.desc, firstI(), "field %d descending", tt.field)
	}
	assert.NoError(t, StructSliceSort(rows, []int{0}, true))
	assert.False(t, rows[0].B, "false first")
	assert.NoError(t, StructSliceSort(rows, []int{6}, true), "a Stringer field")
	assert.Error(t, StructSliceSort(rows, []int{7}, true), "a struct field")
	assert.NoError(t, StructSliceSort([]sortRow{}, []int{1}, true))
}

// FieldsInner is embedded in fieldsOuter.
type FieldsInner struct {
	A int `default:"5"`
	B string
}

// fieldsOuter is a struct for testing the field walking and default functions.
type fieldsOuter struct {
	FieldsInner
	C float32 `default:"1.5" desc:"the c"`
	D bool    `default:"true"`
	E image.Point
	F []int `default:"[1, 2]"`
	g int
}

func TestWalkFields(t *testing.T) {
	var o fieldsOuter
	v := reflect.ValueOf(&o).Elem()
	assert.Equal(t, 6, NumAllFields(v), "A, B, C, D, E, F; not g")

	var names []string
	var parents []string
	WalkFields(v, func(parent reflect.Value, field reflect.StructField, value reflect.Value) bool {
		return field.Name != "E"
	}, func(parent reflect.Value, parentField *reflect.StructField, field reflect.StructField, value reflect.Value) {
		names = append(names, field.Name)
		if parentField != nil {
			parents = append(parents, parentField.Name)
		}
	})
	assert.Equal(t, []string{"A", "B", "C", "D", "F"}, names)
	assert.Equal(t, []string{"FieldsInner", "FieldsInner"}, parents)
}

func TestSetFromDefaultTags(t *testing.T) {
	var o fieldsOuter
	assert.NoError(t, SetFromDefaultTags(&o))
	assert.Equal(t, 5, o.A, "embedded")
	assert.Equal(t, float32(1.5), o.C)
	assert.True(t, o.D)
	assert.Equal(t, []int{1, 2}, o.F)
	assert.NoError(t, SetFromDefaultTags((*fieldsOuter)(nil)))

	var i int
	assert.NoError(t, SetFromDefaultTag(reflect.ValueOf(&i).Elem(), "3:9"), "a range uses its first value")
	assert.Equal(t, 3, i)
	assert.NoError(t, SetFromDefaultTag(reflect.ValueOf(&i).Elem(), ""))
	assert.Equal(t, 3, i, "no default")

	bad := struct {
		X int `default:"x"`
	}{}
	assert.Error(t, SetFromDefaultTags(&bad))
}

func TestValueIsDefault(t *testing.T) {
	assert.True(t, ValueIsDefault(reflect.ValueOf(5), "3:9"))
	assert.False(t, ValueIsDefault(reflect.ValueOf(10), "3:9"))
	assert.True(t, ValueIsDefault(reflect.ValueOf(2), "1,2"))
	assert.False(t, ValueIsDefault(reflect.ValueOf(3), "1,2"))
	assert.True(t, ValueIsDefault(reflect.ValueOf([]int{1, 2}), "[1, 2]"))
	assert.True(t, ValueIsDefault(reflect.ValueOf(0), ""))
	assert.False(t, ValueIsDefault(reflect.ValueOf(0), "x"), "a bad default")
}

func TestStructTags(t *testing.T) {
	assert.Nil(t, StructTags(""))
	tags := StructTags(`json:"a" default:"5" x:`)
	assert.Equal(t, map[string]string{"json": "a", "default": "5"}, tags)
	f, _ := reflect.TypeFor[fieldsOuter]().FieldByName("C")
	assert.Equal(t, "1.5", StructTags(f.Tag)["default"])
}

func TestStringJSON(t *testing.T) {
	assert.Equal(t, "{\n\t\"X\": 1,\n\t\"Y\": 2\n}\n", StringJSON(image.Pt(1, 2)))
}

// TestNonDefaultFieldsUnexported tests that unexported fields are not
// among the non-default fields (and do not panic).
func TestNonDefaultFieldsUnexported(t *testing.T) {
	type withPrivate struct {
		A       int `default:"1"`
		private int
	}
	v := &withPrivate{A: 2, private: 5}
	assert.Equal(t, map[string]any{"A": 2}, NonDefaultFields(v))
}
