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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"goki.dev/goki/config"
	"goki.dev/grog"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

var (
	GOOS   = runtime.GOOS
	GOARCH = runtime.GOARCH
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
	grog.PrintlnInfo("write", filename)

	if err := Mkdir(c, filepath.Dir(filename)); err != nil {
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

func Mkdir(c *config.Config, dir string) error {
	if c.Build.Print || c.Build.PrintOnly {
		PrintCmd("mkdir -p %s", dir)
	}
	if c.Build.PrintOnly {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

func Symlink(c *config.Config, src, dst string) error {
	if c.Build.Print || c.Build.PrintOnly {
		PrintCmd("ln -s %s %s", src, dst)
	}
	if c.Build.PrintOnly {
		return nil
	}
	if GOOS == "windows" {
		return DoCopyAll(dst, src)
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

func RemoveAll(c *config.Config, path string) error {
	if c.Build.Print || c.Build.PrintOnly {
		PrintCmd(`rm -r -f "%s"`, path)
	}
	if c.Build.PrintOnly {
		return nil
	}

	// os.RemoveAll behaves differently in windows.
	// http://golang.org/issues/9606
	if GOOS == "windows" {
		ResetReadOnlyFlagAll(path)
	}

	return os.RemoveAll(path)
}

func ResetReadOnlyFlagAll(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return os.Chmod(path, 0666)
	}
	fd, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fd.Close()

	names, _ := fd.Readdirnames(-1)
	for _, name := range names {
		ResetReadOnlyFlagAll(path + string(filepath.Separator) + name)
	}
	return nil
}

func GoEnv(name string) string {
	if val := os.Getenv(name); val != "" {
		return val
	}
	val, err := exec.Command("go", "env", name).Output()
	if err != nil {
		panic(err) // the Go tool was tested to work earlier
	}
	return strings.TrimSpace(string(val))
}

func RunCmd(c *config.Config, cmd *exec.Cmd) error {
	if c.Build.Print || c.Build.PrintOnly {
		dir := ""
		if cmd.Dir != "" {
			dir = "PWD=" + cmd.Dir + " "
		}
		env := strings.Join(cmd.Env, " ")
		if env != "" {
			env += " "
		}
		PrintCmd("%s%s%s", dir, env, strings.Join(cmd.Args, " "))
	}

	buf := new(bytes.Buffer)
	buf.WriteByte('\n')
	if grease.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = buf
		cmd.Stderr = buf
	}

	if c.Build.Work {
		if GOOS == "windows" {
			cmd.Env = append(cmd.Env, `TEMP=`+TmpDir)
			cmd.Env = append(cmd.Env, `TMP=`+TmpDir)
		} else {
			cmd.Env = append(cmd.Env, `TMPDIR=`+TmpDir)
		}
	}

	if !c.Build.PrintOnly {
		cmd.Env = Environ(cmd.Env)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s failed: %v%s", strings.Join(cmd.Args, " "), err, buf)
		}
	}
	return nil
}
