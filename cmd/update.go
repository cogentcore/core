// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/cmd/config"
	"cogentcore.org/core/cmd/web"
	"golang.org/x/mod/semver"
)

// goTool is a command that is installed with go install.
type goTool struct {

	// name is the name of the command.
	name string

	// pkg is the package of the command.
	pkg string

	// module is the module that holds the package.
	module string

	// query is the version query to update to, such as main or latest.
	query string

	// path is the path of the installed command, or "" if it is not installed.
	path string
}

// Update updates the core command and the tools that it uses to their
// latest versions: the core command itself (from the main branch, as it is
// installed), and the GoScript compiler and bun that the js platform uses, if
// they are installed. A tool built from a local source checkout is not
// replaced unless the force option is set, and the check option only
// reports which tools have updates.
func Update(c *config.Config) error { //types:add
	core, err := os.Executable()
	if err != nil {
		return err
	}
	tools := []goTool{
		{name: "core", pkg: "cogentcore.org/core", module: "cogentcore.org/core", query: "main", path: core},
		{name: "goscript", pkg: web.GoscriptPackage, module: "github.com/s4wave/goscript", query: "latest", path: web.ToolPath("goscript")},
	}
	var errs []error
	for _, t := range tools {
		if err := updateGoTool(c, t); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", t.name, err))
		}
	}
	if err := updateBun(c); err != nil {
		errs = append(errs, fmt.Errorf("bun: %w", err))
	}
	return errors.Join(errs...)
}

// updateGoTool updates the given tool installed with go install, if it is
// installed and a newer version is available.
func updateGoTool(c *config.Config, t goTool) error {
	// The installed version is the version of the main module of the
	// command; a command built from a local source checkout has no sum.
	if t.path == "" {
		fmt.Println(t.name + " is not installed")
		return nil
	}
	info, err := buildinfo.ReadFile(t.path)
	if err != nil {
		return fmt.Errorf("reading the version of %s: %w", t.path, err)
	}
	if info.Main.Path != t.module {
		fmt.Printf("%s (%s) is not built from %s; not updating it\n", t.name, t.path, t.module)
		return nil
	}
	installed := info.Main.Version
	latest, err := latestModuleVersion(t.module, t.query)
	if err != nil {
		return err
	}

	// Only an older version installed from the module proxy is updated
	// without the force option.
	switch {
	case installed == latest:
		fmt.Printf("%s %s is up to date\n", t.name, installed)
		return nil
	case info.Main.Sum == "" && !c.Update.Force:
		fmt.Printf("%s %s was built from a local source checkout; not replacing it with %s (use -force to do so)\n", t.name, installed, latest)
		return nil
	case semver.Compare(installed, latest) > 0 && !c.Update.Force:
		fmt.Printf("%s %s is newer than %s; not replacing it (use -force to do so)\n", t.name, installed, latest)
		return nil
	case c.Update.Check:
		fmt.Printf("%s %s can be updated to %s\n", t.name, installed, latest)
		return nil
	}
	logx.PrintlnWarn(fmt.Sprintf("updating %s from %s to %s", t.name, installed, latest))
	return exec.Verbose().SetBuffer(false).SetDir(os.TempDir()).Run("go", "install", t.pkg+"@"+t.query)
}

// latestModuleVersion returns the version of the given module for the given
// version query (such as main or latest), from the module proxy.
func latestModuleVersion(module, query string) (string, error) {
	// A workspace or the module of the current directory does not apply.
	v, err := exec.Silent().SetDir(os.TempDir()).SetEnv("GOWORK", "off").Output("go", "list", "-m", "-f", "{{.Version}}", module+"@"+query)
	if err != nil {
		return "", fmt.Errorf("finding the %s version of %s: %w", query, module, err)
	}
	return strings.TrimSpace(v), nil
}

// updateBun updates bun with its own upgrade command, if it is installed.
// With the check option, it reports the latest release instead.
func updateBun(c *config.Config) error {
	path := web.ToolPath("bun")
	if path == "" {
		fmt.Println("bun is not installed")
		return nil
	}
	installed, err := exec.Silent().Output(path, "--version")
	if err != nil {
		return err
	}
	installed = strings.TrimSpace(installed)
	if !c.Update.Check {
		logx.PrintlnWarn("updating bun " + installed + " with bun upgrade")
		return exec.Verbose().SetBuffer(false).Run(path, "upgrade")
	}

	// The latest release is only needed to report it.
	latest, err := latestBunVersion()
	switch {
	case err != nil:
		fmt.Printf("bun %s is installed; could not find the latest version: %v\n", installed, err)
	case latest == installed:
		fmt.Printf("bun %s is up to date\n", installed)
	default:
		fmt.Printf("bun %s can be updated to %s\n", installed, latest)
	}
	return nil
}

// latestBunVersion returns the version of the latest release of bun on
// GitHub, whose tags are named bun-v1.2.3.
func latestBunVersion() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/oven-sh/bun/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub returned %s", resp.Status)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return strings.TrimPrefix(release.TagName, "bun-v"), nil
}
