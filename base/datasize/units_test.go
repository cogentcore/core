// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// These tests are not compiled by goscript, which gives untyped constants
// converted to the uint64 Size the type of a JavaScript number instead of
// a bigint, which TypeScript rejects.

//go:build !goscript

package datasize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	for in, want := range map[string]Size{"7 kilobytes": 7 * KB, "3 E": 3 * EB, "1 byte": 1} {
		err := s.UnmarshalText([]byte(in))
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
