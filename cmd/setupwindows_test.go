// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCgoFlagsWith(t *testing.T) {
	obj := `C:/llvm/typeinfo.o`
	assert.Equal(t, "-O2 -g "+obj, cgoFlagsWith("-O2 -g\n", obj))
	assert.Equal(t, "-O2 -g "+obj, cgoFlagsWith("-O2 -g "+obj, obj), "already there")
	assert.Equal(t, obj, cgoFlagsWith("", obj))
	assert.Equal(t, `-O2 -g "C:/Users/A B/typeinfo.o"`, cgoFlagsWith("-O2 -g", "C:/Users/A B/typeinfo.o"))
}

func TestGoFlagsWithTag(t *testing.T) {
	assert.Equal(t, "-tags=exp.winarm64", goFlagsWithTag("", "exp.winarm64"))
	assert.Equal(t, "-mod=mod -tags=exp.winarm64", goFlagsWithTag("-mod=mod\n", "exp.winarm64"))
	assert.Equal(t, "-tags=a,exp.winarm64 -v", goFlagsWithTag("-tags=a -v", "exp.winarm64"))
	assert.Equal(t, "--tags=a,exp.winarm64", goFlagsWithTag("--tags=a", "exp.winarm64"))
	assert.Equal(t, "-tags=exp.winarm64", goFlagsWithTag("-tags=", "exp.winarm64"))
	assert.Equal(t, "-tags=a,exp.winarm64", goFlagsWithTag("-tags=a,exp.winarm64", "exp.winarm64"), "already there")
}

// TestTypeInfoStub tests that the type_info stub defines the MSVC symbol
// that the wgpu_native library needs, if zig is available to compile it
// for Windows ARM64 (with MinGW, as llvm-mingw does).
func TestTypeInfoStub(t *testing.T) {
	zig, err := exec.LookPath("zig")
	if err != nil {
		t.Skip("zig is not available")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "typeinfo.c")
	obj := filepath.Join(dir, "typeinfo.o")
	require.NoError(t, os.WriteFile(src, []byte(typeInfoStub), 0o644))
	out, err := exec.Command(zig, "cc", "-target", "aarch64-windows-gnu", "-c", src, "-o", obj).CombinedOutput()
	require.NoError(t, err, string(out))
	b, err := os.ReadFile(obj)
	require.NoError(t, err)
	assert.Contains(t, string(b), "??_7type_info@@6B@")
}
