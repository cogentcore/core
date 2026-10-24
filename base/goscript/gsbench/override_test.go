// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gsbench

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// Benchmarks of method calls on pointers to types that goscript implements
// by hand (overrides): bytes.Buffer, strings.Builder and time.Time.

func BenchmarkBytesBufferPointer(b *testing.B) {
	buf := &bytes.Buffer{}
	for range b.N {
		buf.Reset()
		for range 100 {
			buf.WriteByte('a')
		}
		sinkI = buf.Len()
	}
}

func BenchmarkStringBuilderPointer(b *testing.B) {
	for range b.N {
		sb := &strings.Builder{}
		for range 100 {
			sb.WriteString("ab")
		}
		sinkI = sb.Len()
	}
}

func BenchmarkTimeMethods(b *testing.B) {
	t0 := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	for range b.N {
		t := t0
		var d time.Duration
		for range 100 {
			t = t.Add(time.Millisecond)
			d += t.Sub(t0)
		}
		sinkI = int(d)
	}
}
