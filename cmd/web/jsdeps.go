// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/exec"
)

// GoscriptModule is the Go module of the GoScript compiler.
const GoscriptModule = "github.com/s4wave/goscript"

// goscriptManifest is the part of the package.json of GoScript that the js
// platform uses: the dependencies are the npm packages that the GoScript
// runtime imports, which the bundle needs, and the development
// dependencies have the version of tsrs that goscript test uses.
type goscriptManifest struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// goscriptPackageJSON returns the package.json of the installed goscript
// command, which comes from the source of its module, so that the npm
// dependencies always match the GoScript runtime in the compiled output.
func goscriptPackageJSON() (*goscriptManifest, error) {
	// The module source is found from the build info of the command.
	bin := ToolPath("goscript")
	if bin == "" {
		return nil, errors.New("goscript is not installed")
	}
	dir, err := goscriptModuleDir(bin)
	if err != nil {
		return nil, err
	}

	// Its package.json lists the npm packages of the runtime.
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("reading the GoScript package.json: %w", err)
	}
	m := &goscriptManifest{}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("parsing the GoScript package.json in %s: %w", dir, err)
	}
	return m, nil
}

// goscriptModuleDir returns the source directory of the GoScript module that
// the given goscript command was built from. A released version is found in
// the module cache, downloading it if needed. A command built from a local
// checkout has no released version, so the module is then looked up in the
// current Go module or workspace, which normally has that checkout.
func goscriptModuleDir(bin string) (string, error) {
	info, err := buildinfo.ReadFile(bin)
	if err != nil {
		return "", fmt.Errorf("reading the build info of %s: %w", bin, err)
	}
	version := info.Main.Version
	var downloadErr error
	if version != "" && version != "(devel)" && !strings.Contains(version, "+dirty") {
		dir, err := goModuleDir(os.TempDir(), "download", GoscriptModule+"@"+version)
		if err == nil {
			return dir, nil
		}
		downloadErr = err
	}
	dir, err := goModuleDir("", "list", GoscriptModule)
	if err == nil {
		return dir, nil
	}
	return "", fmt.Errorf("could not find the source of GoScript %s, which has the npm dependencies of its runtime in its package.json (install a released version with go install %s@latest): %w", version, GoscriptPackage, errors.Join(downloadErr, err))
}

// goModuleDir returns the source directory of the given module from the
// JSON output of go mod download (cmd "download") or go list -m (cmd "list"),
// run in the given directory ("" for the current one).
func goModuleDir(dir, cmd, module string) (string, error) {
	args := []string{"mod", "download", "-json", module}
	if cmd == "list" {
		args = []string{"list", "-m", "-json", module}
	}
	ec := exec.Silent()
	if dir != "" {
		ec = ec.SetDir(dir)
	}
	out, err := ec.Output("go", args...)
	var mod struct {
		Dir   string
		Error any
	}
	if jerr := json.Unmarshal([]byte(out), &mod); jerr == nil && mod.Error != nil {
		return "", fmt.Errorf("go %s: %v", strings.Join(args, " "), mod.Error)
	}
	if err != nil {
		return "", fmt.Errorf("go %s: %w", strings.Join(args, " "), err)
	}
	if mod.Dir == "" {
		return "", fmt.Errorf("go %s: no module directory", strings.Join(args, " "))
	}
	return mod.Dir, nil
}

// jsPackageJSON returns the package.json used for bundling the GoScript
// output, which has the npm dependencies of the given GoScript package.json.
func jsPackageJSON(m *goscriptManifest) ([]byte, error) {
	b, err := json.Marshal(struct {
		Type         string            `json:"type"`
		Private      bool              `json:"private"`
		Dependencies map[string]string `json:"dependencies,omitempty"`
	}{"module", true, m.Dependencies})
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
