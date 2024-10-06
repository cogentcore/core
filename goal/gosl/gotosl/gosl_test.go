// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gotosl

import (
	"os"
	"testing"

	"cogentcore.org/core/cli"
	"github.com/stretchr/testify/assert"
)

// TestTranslate
func TestTranslate(t *testing.T) {
	os.Chdir("testdata")

	opts := cli.DefaultOptions("gosl", "Go as a shader language converts Go code to WGSL WebGPU shader code, which can be run on the GPU through WebGPU.")
	cfg := &Config{}
	cli.Run(opts, cfg, Run)

	exSh, err := os.ReadFile("Compute.golden")
	if err != nil {
		t.Error(err)
		return
	}
	exGosl, err := os.ReadFile("gosl.golden")
	if err != nil {
		t.Error(err)
		return
	}

	gotSh, err := os.ReadFile("shaders/Compute.wgsl")
	if err != nil {
		t.Error(err)
		return
	}
	gotGosl, err := os.ReadFile("gosl.go")
	if err != nil {
		t.Error(err)
		return
	}

	assert.Equal(t, string(exSh), string(gotSh))
	assert.Equal(t, string(exGosl), string(gotGosl))
}
