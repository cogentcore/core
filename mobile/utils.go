// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"goki.dev/goki/config"
	"goki.dev/grog"
	"goki.dev/xe"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

var (
	GOOS   = runtime.GOOS
	GOARCH = runtime.GOARCH
)

func CopyFile(c *config.Config, dst, src string) error {
	return WriteFile(c, dst, func(w io.Writer) error {
		if c.Build.PrintOnly {
			return nil
		}
		f, err := os.Open(src)
		xe.PrintCmd(fmt.Sprintf("cp %s %s", src, dst), err)
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
	grog.PrintlnInfo("write", filename)

	if err := xe.MkdirAll(filepath.Dir(filename), 0755); err != nil {
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
	xc := xe.Minor().SetDir(src).SetEnv("GOOS", PlatformOS(targetPlatform)).SetEnv("GOARCH", targetArch)

	tags := append(c.Build.Tags[:], PlatformTags(targetPlatform)...)

	// TODO(hyangah): probably we don't need to add all the dependencies.
	output, err := xc.Output("go", "list", "-m", "-json", "-tags="+strings.Join(tags, ","), "all")
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
	e := json.NewDecoder(bytes.NewReader([]byte(output)))
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
		out, err := xe.Minor().Output("go", "env", "GOMOD")
		if err != nil {
			AreGoModulesUsedResult.err = err
			return
		}
		outstr := strings.TrimSpace(string(out))
		AreGoModulesUsedResult.used = outstr != ""
	})
	return AreGoModulesUsedResult.used, AreGoModulesUsedResult.err
}

func Symlink(c *config.Config, src, dst string) error {
	xe.PrintCmd(fmt.Sprintf("ln -s %s %s", src, dst), nil)
	if c.Build.PrintOnly {
		return nil
	}
	if GOOS == "windows" {
		return DoCopyAll(dst, src) // TODO: do we need to do this?
	}
	return os.Symlink(src, dst)
}

func DoCopyAll(dst, src string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, errin error) (err error) {
		if errin != nil {
			return errin
		}
		prefixLen := len(src)
		if len(path) > prefixLen {
			prefixLen++ // file separator
		}
		outpath := filepath.Join(dst, path[prefixLen:])
		if info.IsDir() {
			return os.Mkdir(outpath, 0755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(outpath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer func() {
			if errc := out.Close(); err == nil {
				err = errc
			}
		}()
		_, err = io.Copy(out, in)
		return err
	})
}

func GoEnv(name string) string {
	if val := os.Getenv(name); val != "" {
		return val
	}
	val, err := xe.Minor().Output("go", "env", name)
	if err != nil {
		panic(err) // the Go tool was tested to work earlier
	}
	return strings.TrimSpace(string(val))
}

// Major is a replacement for [xe.Major] that also sets the TMP environment
// variables based on the config options, operating system, and [TmpDir].
func Major(c *config.Config) *xe.Config {
	xc := xe.Major()
	if c.Build.Work {
		if GOOS == "windows" {
			xc.SetEnv("TEMP", TmpDir).SetEnv("TMP", TmpDir)
		} else {
			xc.SetEnv("TMPDIR", TmpDir)
		}
	}
	return xc
}
