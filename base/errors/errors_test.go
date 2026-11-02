// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errors

import (
	"bytes"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestSetLogHook(t *testing.T) {
	var got []error
	prev := SetLogHook(func(err error) bool {
		got = append(got, err)
		return true
	})
	err := New("test error")
	Log(err)
	Log1(1, err)
	Log2(1, 2, err)
	Log(nil)
	if len(got) != 3 {
		t.Errorf("hook got %d errors, want 3", len(got))
	}
	if p := SetLogHook(prev); p == nil {
		t.Error("SetLogHook did not return the hook that it replaces")
	}
}

// TestLogCallerInfo tests that the logged errors have the information
// of the code that calls Log, Log1, Log2, and LogDepth.
func TestLogCallerInfo(t *testing.T) {
	var b bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&b, nil)))
	defer slog.SetDefault(prev)
	err := New("test error")
	line := func() string {
		_, file, line, _ := runtime.Caller(1)
		return file + ":" + strconv.Itoa(line) // of the call after it
	}
	for _, log := range []func() string{
		func() string { at := line(); Log(err); return at },
		func() string { at := line(); Log1(1, err); return at },
		func() string { at := line(); Log2(1, 2, err); return at },
		func() string { at := line(); LogDepth(err, 0); return at },
	} {
		b.Reset()
		at := log()
		if !strings.Contains(b.String(), at) {
			t.Errorf("logged %q, want the caller %s", b.String(), at)
		}
	}
}
