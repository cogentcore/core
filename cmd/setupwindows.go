// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
	"github.com/mitchellh/go-homedir"
)

const (
	// llvmMingwVersion is the version of llvm-mingw installed on Windows ARM64.
	llvmMingwVersion = "20261020"

	// llvmMingwDir is where llvm-mingw is installed on Windows ARM64.
	llvmMingwDir = `C:\llvm-mingw`

	// windowsARM64Tag is the build tag that Delve (used by Cogent Code)
	// requires to build on Windows ARM64, which it supports experimentally.
	windowsARM64Tag = "exp.winarm64"
)

// typeInfoStub is C source for a stub of the MSVC C++ runtime symbol
// const type_info::`vftable', which the MSVC-built wgpu_native library for
// Windows ARM64 references (for Rust panics), but MinGW does not have.
// A Rust panic that unwinds may crash instead of being reported.
const typeInfoStub = `// Stub for MSVC's const type_info::vftable, referenced by the
// MSVC-built wgpu_native library for Rust panics, which MinGW lacks.
// Installed by core setup on Windows ARM64.
const void* type_info_vftable[2] __asm__("??_7type_info@@6B@") = {0, 0};
`

// setupWindows installs the C compiler (w64devkit, or llvm-mingw on ARM64)
// and git, which are needed to build Cogent Core apps on Windows.
func setupWindows(vc *exec.Config) error {
	// We must be in the home directory to avoid permission issues with file downloading.
	hd, err := homedir.Dir()
	if err != nil {
		return err
	}
	err = os.Chdir(hd)
	if err != nil {
		return err
	}
	if runtime.GOARCH == "arm64" {
		err = setupWindowsARM64(vc)
	} else {
		err = setupWindowsGCC(vc)
	}
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("git"); err != nil {
		installer := "Git-2.45.2-64-bit.exe"
		url := "https://github.com/git-for-windows/git/releases/download/v2.45.2.windows.1/" + installer
		if runtime.GOARCH == "arm64" {
			installer = "Git-2.56.0.4-arm64.exe"
			url = "https://github.com/git-for-windows/git/releases/download/v2.56.0.windows.4/" + installer
		}
		err := vc.Run("curl", "-OL", url)
		if err != nil {
			return err
		}
		path, err := filepath.Abs(installer)
		if err != nil {
			return err
		}
		err = vc.Run(path)
		if err != nil {
			return err
		}
	} else {
		logx.PrintlnWarn("git already installed")
	}
	return nil
}

// setupWindowsGCC installs the w64devkit gcc compiler (for amd64).
func setupWindowsGCC(vc *exec.Config) error {
	if _, err := exec.LookPath("gcc"); err == nil {
		logx.PrintlnWarn("gcc already installed")
		return nil
	}
	err := vc.Run("curl", "-OL", "https://github.com/skeeto/w64devkit/releases/download/v2.0.0/w64devkit-x64-2.0.0.exe")
	if err != nil {
		return err
	}
	path, err := filepath.Abs("w64devkit-x64-2.0.0.exe")
	if err != nil {
		return err
	}
	err = vc.Run(path, "x", "-oC:", "-aoa")
	if err != nil {
		return err
	}
	return windowsRegistryAddPath(`C:\w64devkit\bin`)
}

// setupWindowsARM64 sets up cgo on Windows ARM64: w64devkit only has
// an x86 compiler, so it installs the aarch64 clang compiler of llvm-mingw,
// sets it as the Go CC and CXX, links a stub needed for the wgpu_native
// library (see [typeInfoStub]), and sets the build tag that Delve needs.
func setupWindowsARM64(vc *exec.Config) error {
	clang, err := installLLVMMingw(vc)
	if err != nil {
		return err
	}
	clangxx := strings.TrimSuffix(clang, ".exe") + "++"
	if strings.HasSuffix(clang, ".exe") {
		clangxx += ".exe"
	}
	err = vc.Run("go", "env", "-w", "CC="+clang, "CXX="+clangxx)
	if err != nil {
		return err
	}

	obj, err := buildTypeInfoStub(vc, clang)
	if err != nil {
		return err
	}
	ldflags, err := exec.Output("go", "env", "CGO_LDFLAGS")
	if err != nil {
		return err
	}
	if nf := cgoFlagsWith(ldflags, obj); nf != ldflags {
		err = vc.Run("go", "env", "-w", "CGO_LDFLAGS="+nf)
		if err != nil {
			return err
		}
	}

	goflags, err := exec.Output("go", "env", "GOFLAGS")
	if err != nil {
		return err
	}
	if nf := goFlagsWithTag(goflags, windowsARM64Tag); nf != goflags {
		err = vc.Run("go", "env", "-w", "GOFLAGS="+nf)
		if err != nil {
			return err
		}
	}
	return nil
}

// installLLVMMingw installs llvm-mingw for ARM64 hosts in [llvmMingwDir],
// unless its aarch64 clang compiler is already on the path or there,
// and returns the path of that compiler.
func installLLVMMingw(vc *exec.Config) (string, error) {
	const name = "aarch64-w64-mingw32-clang"
	if clang, err := exec.LookPath(name); err == nil {
		logx.PrintlnWarn("llvm-mingw already installed:", clang)
		return clang, nil
	}
	bin := filepath.Join(llvmMingwDir, "bin")
	clang := filepath.Join(bin, name+".exe")
	if _, err := os.Stat(clang); err == nil {
		logx.PrintlnWarn("llvm-mingw already installed in", llvmMingwDir)
		return clang, windowsRegistryAddPath(bin)
	}
	dist := "llvm-mingw-" + llvmMingwVersion + "-ucrt-aarch64"
	err := vc.Run("curl", "-OL", "https://github.com/mstorsjo/llvm-mingw/releases/download/"+llvmMingwVersion+"/"+dist+".zip")
	if err != nil {
		return "", err
	}
	// tar (bsdtar) is included in Windows, and extracts zip files
	err = vc.Run("tar", "-xf", dist+".zip", "-C", filepath.Dir(llvmMingwDir))
	if err != nil {
		return "", err
	}
	err = os.Rename(filepath.Join(filepath.Dir(llvmMingwDir), dist), llvmMingwDir)
	if err != nil {
		return "", err
	}
	return clang, windowsRegistryAddPath(bin)
}

// buildTypeInfoStub compiles [typeInfoStub] with the given clang compiler,
// in the Cogent Core directory of the user config directory, and returns
// the path of the object file.
func buildTypeInfoStub(vc *exec.Config, clang string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "cogentcore", "windows-arm64")
	err = os.MkdirAll(dir, 0o755)
	if err != nil {
		return "", err
	}
	src := filepath.Join(dir, "typeinfo.c")
	err = os.WriteFile(src, []byte(typeInfoStub), 0o644)
	if err != nil {
		return "", err
	}
	obj := filepath.Join(dir, "typeinfo.o")
	return obj, vc.Run(clang, "-c", src, "-o", obj)
}

// cgoFlagsWith returns the given CGO_*FLAGS value with the given flag added
// at the end (quoted if it has a space), unless it already has it.
func cgoFlagsWith(flags, flag string) string {
	flags = strings.TrimSpace(flags)
	if strings.Contains(flags, flag) {
		return flags
	}
	if strings.ContainsAny(flag, " \t") {
		flag = `"` + flag + `"`
	}
	if flags == "" {
		return flag
	}
	return flags + " " + flag
}

// goFlagsWithTag returns the given GOFLAGS value with the given build tag
// added, to an existing -tags flag if there is one, unless it already has it.
func goFlagsWithTag(goflags, tag string) string {
	fields := strings.Fields(goflags)
	for i, f := range fields {
		tags, ok := strings.CutPrefix(f, "-tags=")
		if !ok {
			tags, ok = strings.CutPrefix(f, "--tags=")
		}
		if !ok {
			continue
		}
		for _, t := range strings.Split(tags, ",") {
			if t == tag {
				return strings.Join(fields, " ")
			}
		}
		if tags == "" {
			fields[i] = f + tag
		} else {
			fields[i] = f + "," + tag
		}
		return strings.Join(fields, " ")
	}
	return strings.Join(append(fields, "-tags="+tag), " ")
}
