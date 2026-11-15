// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"errors"
	"image"
	"image/color"
	"testing"
	"time"

	"cogentcore.org/core/colors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MoreInner struct {
	A int
	B []string
}

type MoreOuter struct {
	*MoreInner
	Inner MoreInner
	Items []MoreInner
	Name  string
}

// TestCopyEmbeddedPointer tests deeply copying a struct that embeds a
// pointer to a struct with fields that are not shallow, and that the
// elements of a longer destination slice beyond the source are kept.
func TestCopyEmbeddedPointer(t *testing.T) {
	src := &MoreOuter{MoreInner: &MoreInner{A: 1, B: []string{"x"}}, Inner: MoreInner{A: 2}, Items: []MoreInner{{A: 3}}, Name: "src"}
	dst := &MoreOuter{Items: []MoreInner{{A: 9}, {A: 10}}}
	CopyStructDeep(dst, src)
	require.NotNil(t, dst.MoreInner)
	assert.NotSame(t, src.MoreInner, dst.MoreInner)
	assert.Equal(t, []string{"x"}, dst.B)
	src.B[0] = "changed"
	assert.Equal(t, "x", dst.B[0], "a deep copy")
	assert.Equal(t, 2, dst.Inner.A)
	assert.Equal(t, 3, dst.Items[0].A)
	assert.Len(t, dst.Items, 2, "dst elements beyond src are kept (as documented)")
	assert.Equal(t, "src", dst.Name)
}

// TestCopyFieldsPaths tests copying named fields and sub-fields, and errors.
func TestCopyFieldsPaths(t *testing.T) {
	src := MoreOuter{Inner: MoreInner{A: 5, B: []string{"b"}}, Name: "n"}
	var dst MoreOuter
	require.NoError(t, CopyFields(&dst, &src, "Name", "Inner.A"))
	assert.Equal(t, "n", dst.Name)
	assert.Equal(t, 5, dst.Inner.A)
	assert.Nil(t, dst.Inner.B, "not copied")
	assert.Error(t, CopyFields(&dst, &src, "Nope"))
	assert.Error(t, CopyFields(&dst, &src, "Inner.Nope"))
	assert.Error(t, CopyFields(1, &src, "Name"))
	assert.Error(t, CopyFields(&dst, 1, "Name"))
}

type colorer struct{ c color.RGBA }

func (c *colorer) SetColor(cl color.Color) { c.c = colors.AsRGBA(cl) }

type anyer struct{ v any }

func (a *anyer) SetAny(v any) error {
	if v == "bad" {
		return errors.New("bad")
	}
	a.v = v
	return nil
}

// TestSetRobustMore tests setting values of many kinds robustly.
func TestSetRobustMore(t *testing.T) {
	var img image.Image
	u := colors.Uniform(colors.Red)
	require.NoError(t, SetRobust(&img, u))
	assert.Same(t, u, img, "images are set as pointers")

	var c color.RGBA
	require.NoError(t, SetRobust(&c, "blue"))
	assert.Equal(t, colors.Blue, c)
	un := image.NewUniform(color.Black)
	require.NoError(t, SetRobust(un, "red"))
	assert.Equal(t, color.Color(colors.Red), un.C)
	var cl colorer
	require.NoError(t, SetRobust(&cl, "#00ff00"))
	assert.Equal(t, color.RGBA{0, 255, 0, 255}, cl.c)
	var im2 image.Image
	require.NoError(t, SetRobust(&im2, "green"))
	assert.NotNil(t, im2)

	var a anyer
	require.NoError(t, SetRobust(&a, 3))
	assert.Equal(t, 3, a.v)
	assert.Error(t, SetRobust(&a, "bad"))

	var d time.Duration
	require.NoError(t, SetRobust(&d, "2s"))
	assert.Equal(t, 2*time.Second, d)
	assert.Error(t, SetRobust(&d, "x"))

	var i8 int8
	require.NoError(t, SetRobust(&i8, "7"))
	assert.Equal(t, int8(7), i8)
	assert.Error(t, SetRobust(&i8, "x"))
	var u16 uint16
	require.NoError(t, SetRobust(&u16, 9.0))
	assert.Equal(t, uint16(9), u16)

	var ip *int
	assert.Error(t, SetRobust(ip, 1), "a nil destination")
	pp := &ip
	require.NoError(t, SetRobust(pp, 4), "a pointer to a nil pointer")
	var s string
	require.NoError(t, SetRobust(&s, nil), "a nil source does nothing")
}

// TestToStringPointers tests converting pointers and other values to
// strings.
func TestToStringPointers(t *testing.T) {
	s, b, i, i32, i64 := "s", true, 1, int32(2), int64(3)
	by := []byte("by")
	var u8 uint8 = 'u'
	f64, f32 := 1.5, float32(2.5)
	f := false
	assert.Equal(t, "s", ToString(&s))
	assert.Equal(t, "by", ToString(&by))
	assert.Equal(t, "by", ToString(by))
	assert.Equal(t, "true", ToString(&b))
	assert.Equal(t, "false", ToString(&f))
	assert.Equal(t, "false", ToString(f))
	assert.Equal(t, "1", ToString(&i))
	assert.Equal(t, "2", ToString(&i32))
	assert.Equal(t, "2", ToString(i32))
	assert.Equal(t, "3", ToString(&i64))
	assert.Equal(t, "u", ToString(&u8))
	assert.Equal(t, "u", ToString(u8))
	assert.Equal(t, "1.5", ToString(&f64))
	assert.Equal(t, "2.5", ToString(&f32))
	assert.Equal(t, "2.5", ToString(f32))
	var np *int
	assert.Equal(t, "nil", ToString(np))
	assert.Equal(t, "nil", ToString(nil))
	assert.Equal(t, "{1 []}", ToString(MoreInner{A: 1}))
	assert.Equal(t, "2s", ToString(2*time.Second), "a Stringer")
}
