// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
)

// goscriptPackage is the Go package of the GoScript compiler command.
const goscriptPackage = "github.com/s4wave/goscript/cmd/goscript"

// ensureJSTools makes sure that the goscript and bun commands that the js
// platform needs are available, installing them if they are not. GoScript is
// installed with go install, and bun with its official install script for
// the current platform. The directories they are installed into are added
// to the PATH of this process, so that they can be run right away.
func ensureJSTools() error {
	err := ensureTool("goscript", goBinDirs(), installGoscript)
	if err != nil {
		return err
	}
	return ensureTool("bun", bunBinDirs(), installBun)
}

// ensureTool makes sure that the given tool can be found on the PATH. If it
// is not on the PATH, but is in one of the given directories where it is
// normally installed, that directory is added to the PATH. Otherwise, the
// tool is installed with install and looked for again.
func ensureTool(tool string, dirs []string, install func() error) error {
	if _, err := exec.LookPath(tool); err == nil {
		return nil
	}

	// The tool may be installed in a directory that is not on the PATH.
	if addToolDir(tool, dirs) {
		return nil
	}

	// Otherwise, it is installed and must then be found.
	logx.PrintlnWarn("installing " + tool + ", which the js platform needs")
	err := install()
	if err != nil {
		return fmt.Errorf("error installing %s: %w", tool, err)
	}
	if _, err := exec.LookPath(tool); err == nil || addToolDir(tool, dirs) {
		return nil
	}
	return fmt.Errorf("installed %s, but could not find it in %s", tool, strings.Join(dirs, ", "))
}

// addToolDir adds the first of the given directories that contains the
// given tool to the PATH of this process, and returns whether it found one.
func addToolDir(tool string, dirs []string) bool {
	name := tool
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	for _, dir := range dirs {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			continue
		}
		path := dir + string(os.PathListSeparator) + os.Getenv("PATH")
		os.Setenv("PATH", path)
		return true
	}
	return false
}

// goBinDirs returns the directories that go install puts commands in:
// GOBIN if it is set, and otherwise the bin directory of GOPATH.
func goBinDirs() []string {
	if gobin := goEnv("GOBIN"); gobin != "" {
		return []string{gobin}
	}
	if gopath := goEnv("GOPATH"); gopath != "" {
		return []string{filepath.Join(strings.Split(gopath, string(os.PathListSeparator))[0], "bin")}
	}
	return nil
}

// goEnv returns the value of the given go env variable, or "" if it is
// not set or go env fails.
func goEnv(name string) string {
	value, err := exec.Silent().Output("go", "env", name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

// bunBinDirs returns the directories that bun's install script puts bun in:
// the bin directory of BUN_INSTALL if it is set, and otherwise ~/.bun/bin.
func bunBinDirs() []string {
	if bi := os.Getenv("BUN_INSTALL"); bi != "" {
		return []string{filepath.Join(bi, "bin")}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, ".bun", "bin")}
}

// installGoscript installs the latest GoScript compiler with go install.
func installGoscript() error {
	return exec.Verbose().SetBuffer(false).Run("go", "install", goscriptPackage+"@latest")
}

// installBun installs bun with its official install script: a shell script
// on macOS and Linux, and a PowerShell script on Windows.
func installBun() error {
	vc := exec.Verbose().SetBuffer(false)
	if runtime.GOOS == "windows" {
		return vc.Run("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "irm bun.sh/install.ps1 | iex")
	}
	return vc.Run("bash", "-c", "curl -fsSL https://bun.sh/install | bash")
}
