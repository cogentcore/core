// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var update = flag.Bool("update", false, "update .golden files")

func runTest(t *testing.T, in, out string) {
	// process flags
	_, err := os.Lstat(in)
	if err != nil {
		t.Error(err)
		return
	}

	sls, err := ProcessFiles([]string{in})
	if err != nil {
		t.Error(err)
		return
	}

	expected, err := os.ReadFile(out)
	if err != nil {
		t.Error(err)
		return
	}

	var got []byte
	for _, b := range sls {
		got = b
		break
	}

	if !bytes.Equal(got, expected) {
		if *update {
			if in != out {
				if err := os.WriteFile(out, got, 0666); err != nil {
					t.Error(err)
				}
				return
			}
			// in == out: don't accidentally destroy input
			t.Errorf("WARNING: -update did not rewrite input file %s", in)
		}

		assert.Equal(t, string(expected), string(got))
		if err := os.WriteFile(in+".gosl", got, 0666); err != nil {
			t.Error(err)
		}
	}
}

// TestRewrite processes testdata/*.input files and compares them to the
// corresponding testdata/*.golden files. The gosl flags used to process
// a file must be provided via a comment of the form
//
//	//gosl flags
//
// in the processed file within the first 20 lines, if any.
func TestRewrite(t *testing.T) {
	if gomod := os.Getenv("GO111MODULE"); gomod == "off" {
		t.Error("gosl only works in go modules mode, but GO111MODULE=off")
		return
	}

	// determine input files
	match, err := filepath.Glob("testdata/*.go")
	if err != nil {
		t.Fatal(err)
	}

	if *outDir != "" {
		os.MkdirAll(*outDir, 0755)
	}

	for _, in := range match {
		name := filepath.Base(in)
		t.Run(name, func(t *testing.T) {
			out := in // for files where input and output are identical
			if strings.HasSuffix(in, ".go") {
				out = in[:len(in)-len(".go")] + ".golden"
			}
			runTest(t, in, out)
		})
	}
}
