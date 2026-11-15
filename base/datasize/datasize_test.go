// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/c2h5oh/datasize
// Copyright (c) 2016 Maciej Lisiewski

package datasize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalText(t *testing.T) {
	table := []struct {
		in  Size
		out string
	}{
		{0, "0B"},
		{B, "1B"},
		{KB, "1KB"},
		{MB, "1MB"},
		{GB, "1GB"},
		{TB, "1TB"},
		{PB, "1PB"},
		{EB, "1EB"},
		{400 * TB, "400TB"},
		{2048 * MB, "2GB"},
		{B + KB, "1025B"},
		{MB + 20*KB, "1044KB"},
		{100*MB + KB, "102401KB"},
	}

	for _, tt := range table {
		b, _ := tt.in.MarshalText()
		s := string(b)

		if s != tt.out {
			t.Errorf("MarshalText(%d) => %s, want %s", tt.in, s, tt.out)
		}
	}
}

func TestUnmarshalText(t *testing.T) {
	table := []struct {
		in  string
		err bool
		out Size
	}{
		{"0", false, 0},
		{"0B", false, 0},
		{"0 KB", false, 0},
		{"1", false, B},
		{"1K", false, KB},
		{"2MB", false, 2 * MB},
		{"5 GB", false, 5 * GB},
		{"20480 G", false, 20 * TB},
		{"50 eB", true, (1 << 64) - 1},
		{"200000 pb", true, (1 << 64) - 1},
		{"10 Mb", true, 0},
		{"g", true, 0},
		{"10 kB ", false, 10 * KB},
		{"10 kBs ", true, 0},
	}

	for _, tt := range table {
		t.Run("UnmarshalText "+tt.in, func(t *testing.T) {
			var s Size
			err := s.UnmarshalText([]byte(tt.in))

			if (err != nil) != tt.err {
				t.Errorf("UnmarshalText(%s) => %v, want no error", tt.in, err)
			}

			if s != tt.out {
				t.Errorf("UnmarshalText(%s) => %d bytes, want %d bytes", tt.in, s, tt.out)
			}
		})
		t.Run("Parse "+tt.in, func(t *testing.T) {
			s, err := Parse([]byte(tt.in))

			if (err != nil) != tt.err {
				t.Errorf("Parse(%s) => %v, want no error", tt.in, err)
			}

			if s != tt.out {
				t.Errorf("Parse(%s) => %d bytes, want %d bytes", tt.in, s, tt.out)
			}
		})
		t.Run("MustParse "+tt.in, func(t *testing.T) {
			defer func() {
				if err := recover(); (err != nil) != tt.err {
					t.Errorf("MustParse(%s) => %v, want no error", tt.in, err)
				}
			}()

			s := MustParse([]byte(tt.in))
			if s != tt.out {
				t.Errorf("MustParse(%s) => %d bytes, want %d bytes", tt.in, s, tt.out)
			}
		})
		t.Run("ParseString "+tt.in, func(t *testing.T) {
			s, err := ParseString(tt.in)

			if (err != nil) != tt.err {
				t.Errorf("ParseString(%s) => %v, want no error", tt.in, err)
			}

			if s != tt.out {
				t.Errorf("ParseString(%s) => %d bytes, want %d bytes", tt.in, s, tt.out)
			}
		})
		t.Run("MustParseString "+tt.in, func(t *testing.T) {
			defer func() {
				if err := recover(); (err != nil) != tt.err {
					t.Errorf("MustParseString(%s) => %v, want no error", tt.in, err)
				}
			}()

			s := MustParseString(tt.in)
			if s != tt.out {
				t.Errorf("MustParseString(%s) => %d bytes, want %d bytes", tt.in, s, tt.out)
			}
		})
	}
}

func TestSizeUnits(t *testing.T) {
	assert.Equal(t, uint64(1536), (KB + 512).Bytes())
	assert.Equal(t, 1.5, (KB + 512).KBytes())
	assert.Equal(t, 2.5, (2*MB + 512*KB).MBytes())
	assert.Equal(t, 1.25, (GB + 256*MB).GBytes())
	assert.Equal(t, 3.0, (3 * TB).TBytes())
	assert.Equal(t, 0.5, (512 * TB).PBytes())
	assert.Equal(t, 2.0, (2 * EB).EBytes())

	tests := []struct {
		s    Size
		want string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{KB + 512, "1.5 KB"},
		{3 * MB / 2, "1.5 MB"},
		{5*GB + GB/2, "5.5 GB"},
		{2*TB + 1, "2.0 TB"},
		{3*PB + PB/10, "3.1 PB"},
		{3*EB + EB/2, "3.5 EB"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.s.String(), "%d", uint64(tt.s))
	}
}

func TestUnmarshalTextErrors(t *testing.T) {
	var s Size
	assert.NoError(t, s.UnmarshalText(nil), "empty is zero")
	assert.Equal(t, Size(0), s)
	for _, bad := range []string{"kb", "12 parsecs", "5Mb", "99999999999999999999", "20000000EB", "20000000000PB"} {
		assert.Error(t, s.UnmarshalText([]byte(bad)), "%q", bad)
	}
	for in, want := range map[string]Size{"7 kilobytes": 7 * KB, "2exa": 0, "3 E": 3 * EB, "1 byte": 1} {
		err := s.UnmarshalText([]byte(in))
		if want == 0 {
			continue // not a unit name
		}
		assert.NoError(t, err, in)
		assert.Equal(t, want, s, in)
	}
}

// TestExaAndExactUnits tests the exabyte unit names, and that sizes of
// exactly a unit are shown in that unit.
func TestExaAndExactUnits(t *testing.T) {
	for _, u := range []string{"2exa", "2exabyte", "2 exabytes", "2EB", "2e"} {
		var s Size
		assert.NoError(t, s.UnmarshalText([]byte(u)), u)
		assert.Equal(t, 2*EB, s, u)
	}
	assert.Equal(t, "1.0 KB", KB.String())
	assert.Equal(t, "1.0 MB", MB.String())
	assert.Equal(t, "1.0 EB", EB.String())
	assert.Equal(t, "1023 B", Size(1023).String())
}
