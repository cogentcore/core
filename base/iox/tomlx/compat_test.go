// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tomlx

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/goscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pelletierFile is the file written from compatValue by the
// pelletier/go-toml/v2 based tomlx (see testdata/pelletier/main.go).
var pelletierFile = filepath.Join("testdata", "pelletier.toml")

// expectedCompat returns compatValue as it is read back from a file:
// nil slices are written as empty arrays.
func expectedCompat() *compatData {
	v := compatValue()
	v.NilSlice = []string{}
	return v
}

// assertCompat asserts that the given values are equal.
// Under goscript, it compares their encodings instead, as its
// reflect.DeepEqual does not yet treat values made by reflect the same
// as the others, and it skips the fields with generic struct types,
// which its reflect reports as having type any.
func assertCompat(t *testing.T, want, got *compatData) {
	t.Helper()
	if !goscript.Enabled {
		assert.Equal(t, want, got)
		return
	}
	for _, v := range []*compatData{want, got} {
		v.Option = compatOption[[]compatMapsItem]{}
		v.List = compatList[string, *compatInner]{}
	}
	wb, err := WriteBytes(want)
	require.NoError(t, err)
	gb, err := WriteBytes(got)
	require.NoError(t, err)
	assert.Equal(t, string(wb), string(gb))
}

// TestReadPelletier tests reading a file written by the previous
// pelletier-based version, which has durations as integers and maps
// with integer and enum keys.
func TestReadPelletier(t *testing.T) {
	var v compatData
	require.NoError(t, Open(&v, pelletierFile))
	assertCompat(t, expectedCompat(), &v)
}

// TestWritePelletier tests that the encoding is identical to that of
// the previous pelletier-based version, so that older versions can
// read the files and they do not change when saved again.
func TestWritePelletier(t *testing.T) {
	want, err := os.ReadFile(pelletierFile)
	require.NoError(t, err)
	b, err := WriteBytes(compatValue())
	require.NoError(t, err)
	assert.Equal(t, string(want), string(b))
}

// TestRoundTrip tests that values read back the same as written.
func TestRoundTrip(t *testing.T) {
	b, err := WriteBytes(compatValue())
	require.NoError(t, err)
	var v compatData
	require.NoError(t, ReadBytes(&v, b))
	assertCompat(t, expectedCompat(), &v)

	// the settings save code encodes a map of fields
	m := map[string]any{"Dur": 2 * time.Second, "IntMap": map[int]string{3: "c"}, "Enum": enumApple, "Inners": compatValue().Inners}
	b, err = WriteBytes(m)
	require.NoError(t, err)
	v = compatData{}
	require.NoError(t, ReadBytes(&v, b))
	assert.Equal(t, 2*time.Second, v.Dur)
	assert.Equal(t, map[int]string{3: "c"}, v.IntMap)
	assert.Equal(t, enumApple, v.Enum)
	assert.Equal(t, compatValue().Inners, v.Inners)
}

// burntSushiFile is the file written from burntSushiValue by
// BurntSushi/toml v1.6.0, which the goscript branch used for a while.
var burntSushiFile = filepath.Join("testdata", "burntsushi.toml")

// burntSushiValue returns compatValue without the maps that
// BurntSushi/toml cannot encode (those with non-string keys).
func burntSushiValue() *compatData {
	v := compatValue()
	v.IntMap, v.Int64Map, v.UintMap, v.EnumMap, v.EnumPtrMap, v.IntBoolMap, v.NestedMap = nil, nil, nil, nil, nil, nil, nil
	return v
}

// TestReadBurntSushi tests reading a file written by BurntSushi/toml:
// durations are strings such as "5s", strings are double-quoted,
// arrays are inline, and nil slices are omitted.
func TestReadBurntSushi(t *testing.T) {
	var v compatData
	require.NoError(t, Open(&v, burntSushiFile))
	want := burntSushiValue()
	want.EmptyMap = map[string]int{}
	assertCompat(t, want, &v)
}

// TestMerge tests that reading into an existing value only changes
// the values in the file, so that files are applied on top of defaults.
func TestMerge(t *testing.T) {
	v := compatValue()
	require.NoError(t, ReadBytes(v, []byte("Int = 5\n[IntMap]\n  2 = 'two'\n[Inner]\n  Name = 'new'\n")))
	want := compatValue()
	want.Int = 5
	want.IntMap[2] = "two"
	want.Inner.Name = "new"
	assert.Equal(t, want, v)

	// embedded structs can also be set by their type name
	require.NoError(t, ReadBytes(v, []byte("[CompatEmbed]\n  EmbedName = 'by name'\n")))
	assert.Equal(t, "by name", v.EmbedName)
	assert.Equal(t, 7, v.EmbedInt)
}

// TestDecodeVariants tests decoding values in forms other than
// those written by the encoder.
func TestDecodeVariants(t *testing.T) {
	var v compatData
	data := `
string = 'lower case key'
Dur = '1m30s'
Durs = ['1ms', 2000]
Float32 = 3
Int = 4.0
Enum = 'Apple'
Bytes = 'abc'
Floats = [1.0, 2.0]
AnyValue = [{A = 1}, {B = 2}]
[[AnyMap.List]]
  X = 1
`
	require.NoError(t, ReadBytes(&v, []byte(data)))
	assert.Equal(t, "lower case key", v.String)
	assert.Equal(t, 90*time.Second, v.Dur)
	assert.Equal(t, []time.Duration{time.Millisecond, 2 * time.Microsecond}, v.Durs)
	assert.Equal(t, float32(3), v.Float32)
	assert.Equal(t, 4, v.Int)
	assert.Equal(t, enumApple, v.Enum)
	assert.Equal(t, []byte("abc"), v.Bytes)
	assert.Equal(t, [4]float32{1, 2, 0, 0}, v.Floats)
	assert.Equal(t, []any{map[string]any{"A": int64(1)}, map[string]any{"B": int64(2)}}, v.AnyValue)
	assert.Equal(t, []any{map[string]any{"X": int64(1)}}, v.AnyMap["List"])
}

// TestDecodeErrors tests that invalid data returns errors with the key path.
func TestDecodeErrors(t *testing.T) {
	tests := []struct {
		data string
		err  string
	}{
		{"Int = 'x'", "Int: cannot decode TOML string into int"},
		{"Int8 = 300", "Int8: 300 overflows int8"},
		{"Uint8 = -1", "Uint8: -1 overflows uint8"},
		{"Dur = 'xyz'", `Dur: time: invalid duration "xyz"`},
		{"Enum = 'Cherry'", "Enum: invalid compatEnum: Cherry"},
		{"[IntMap]\n  x = 'y'", `IntMap: map key "x"`},
		{"[Inner]\n  Tags = 1", "Inner.Tags: cannot decode TOML integer into []string"},
		{"Int = ", "toml: line 1"},
	}
	for _, tt := range tests {
		var v compatData
		err := ReadBytes(&v, []byte(tt.data))
		if assert.Error(t, err, tt.data) {
			assert.Contains(t, err.Error(), tt.err)
		}
	}
}

// TestEncodeErrors tests values that cannot be encoded.
func TestEncodeErrors(t *testing.T) {
	_, err := WriteBytes(42)
	assert.Error(t, err)
	_, err = WriteBytes(map[string]uint64{"x": 1 << 63})
	assert.Error(t, err)
	_, err = WriteBytes(map[string][]any{"x": {nil}})
	assert.Error(t, err)
	_, err = WriteBytes(map[complex64]int{1: 1})
	assert.Error(t, err)
}

// TestEncodeFormat tests the formatting of special values.
func TestEncodeFormat(t *testing.T) {
	type s struct {
		Nested    [][]int
		Empty     []compatInner
		Func      func()
		Omit      int `toml:",omitempty"`
		Inline    []any
		Key       map[string]int
		NilIface  any
		ZeroFloat float64
	}
	b, err := WriteBytes(&s{Nested: [][]int{{1, 2}, {3}}, Inline: []any{map[string]int{"A": 1}, 2}, Key: map[string]int{"": 1, "a'b": 2, "é": 3}})
	require.NoError(t, err)
	want := `Nested = [
  [1, 2],
  [3]
]
Empty = []
Inline = [
  {A = 1},
  2
]
ZeroFloat = 0.0

[Key]
  '' = 1
  "a'b" = 2
  'é' = 3
`
	assert.Equal(t, want, string(b))
	var v s
	require.NoError(t, ReadBytes(&v, b))
	assert.Equal(t, [][]int{{1, 2}, {3}}, v.Nested)
	assert.Equal(t, 3, v.Key["é"])
}
