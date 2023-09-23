// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"goki.dev/goki/config"
	"goki.dev/grease"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

func CopyFile(c *config.Config, dst, src string) error {
	if c.Build.Print {
		PrintCmd("cp %s %s", src, dst)
	}
	return WriteFile(c, dst, func(w io.Writer) error {
		if c.Build.PrintOnly {
			return nil
		}
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(w, f); err != nil {
			return fmt.Errorf("cp %s %s failed: %v", src, dst, err)
		}
		return nil
	})
}

func WriteFile(c *config.Config, filename string, generate func(io.Writer) error) error {
	if grease.Verbose {
		fmt.Fprintf(os.Stderr, "write %s\n", filename)
	}

	if err := mkdir(filepath.Dir(filename)); err != nil {
		return err
	}

	if c.Build.PrintOnly {
		return generate(io.Discard)
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()

	return generate(f)
}

func PackagesConfig(c *config.Config, t *config.Platform) *packages.Config {
	config := &packages.Config{}
	// Add CGO_ENABLED=1 explicitly since Cgo is disabled when GOOS is different from host OS.
	config.Env = append(os.Environ(), "GOARCH="+t.Arch, "GOOS="+PlatformOS(t.OS), "CGO_ENABLED=1")
	tags := append(c.Build.Tags[:], PlatformTags(t.OS)...)

	if len(tags) > 0 {
		config.BuildFlags = []string{"-tags=" + strings.Join(tags, ",")}
	}
	return config
}

// GetModuleVersions returns a module information at the directory src.
func GetModuleVersions(c *config.Config, targetPlatform string, targetArch string, src string) (*modfile.File, error) {
	cmd := exec.Command("go", "list")
	cmd.Env = append(os.Environ(), "GOOS="+PlatformOS(targetPlatform), "GOARCH="+targetArch)

	tags := append(c.Build.Tags[:], PlatformTags(targetPlatform)...)

	// TODO(hyangah): probably we don't need to add all the dependencies.
	cmd.Args = append(cmd.Args, "-m", "-json", "-tags="+strings.Join(tags, ","), "all")
	cmd.Dir = src

	output, err := cmd.Output()
	if err != nil {
		// Module information is not available at src.
		return nil, nil
	}

	type Module struct {
		Main    bool
		Path    string
		Version string
		Dir     string
		Replace *Module
	}

	f := &modfile.File{}
	f.AddModuleStmt("gobind")
	e := json.NewDecoder(bytes.NewReader(output))
	for {
		var mod *Module
		err := e.Decode(&mod)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if mod != nil {
			if mod.Replace != nil {
				p, v := mod.Replace.Path, mod.Replace.Version
				if modfile.IsDirectoryPath(p) {
					// replaced by a local directory
					p = mod.Replace.Dir
				}
				f.AddReplace(mod.Path, mod.Version, p, v)
			} else {
				// When the version part is empty, the module is local and mod.Dir represents the location.
				if v := mod.Version; v == "" {
					f.AddReplace(mod.Path, mod.Version, mod.Dir, "")
				} else {
					f.AddRequire(mod.Path, v)
				}
			}
		}
		if err == io.EOF {
			break
		}
	}
	return f, nil
}

// WriteGoMod writes go.mod file at dir when Go modules are used.
func WriteGoMod(c *config.Config, dir, targetPlatform, targetArch string) error {
	m, err := AreGoModulesUsed()
	if err != nil {
		return err
	}
	// If Go modules are not used, go.mod should not be created because the dependencies might not be compatible with Go modules.
	if !m {
		return nil
	}

	return WriteFile(c, filepath.Join(dir, "go.mod"), func(w io.Writer) error {
		f, err := GetModuleVersions(c, targetPlatform, targetArch, ".")
		if err != nil {
			return err
		}
		if f == nil {
			return nil
		}
		bs, err := f.Format()
		if err != nil {
			return err
		}
		if _, err := w.Write(bs); err != nil {
			return err
		}
		return nil
	})
}

var (
	AreGoModulesUsedResult struct {
		used bool
		err  error
	}
	AreGoModulesUsedOnce sync.Once
)

func AreGoModulesUsed() (bool, error) {
	AreGoModulesUsedOnce.Do(func() {
		out, err := exec.Command("go", "env", "GOMOD").Output()
		if err != nil {
			AreGoModulesUsedResult.err = err
			return
		}
		outstr := strings.TrimSpace(string(out))
		AreGoModulesUsedResult.used = outstr != ""
	})
	return AreGoModulesUsedResult.used, AreGoModulesUsedResult.err
}
