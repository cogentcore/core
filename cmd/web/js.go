// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/cmd/config"
)

// IsJS returns whether the given configuration is building for the js
// platform, which compiles the app to JavaScript using GoScript
// (https://github.com/s4wave/goscript) instead of to WebAssembly.
// Otherwise, the js platform produces the same static website as web.
func IsJS(c *config.Config) bool {
	return len(c.Build.Target) > 0 && c.Build.Target[0].OS == "js"
}

// jsPackageJSON is the package.json used for bundling the GoScript output,
// which contains the npm dependencies of the GoScript runtime, as listed in
// the dependencies of GoScript's own package.json: its crypto packages use
// the @noble libraries and its compression packages use pako.
const jsPackageJSON = `{"type":"module","private":true,"dependencies":{"@noble/ciphers":"2.2.0","@noble/hashes":"2.2.0","pako":"3.0.2"}}
`

// jsTSConfig is the tsconfig.json used for bundling the GoScript output,
// which maps the @goscript imports to the compiled output.
const jsTSConfig = `{
  "compilerOptions": {
    "target": "ES2022", "module": "ESNext", "moduleResolution": "Bundler",
    "lib": ["ESNext", "DOM"], "strict": true, "allowImportingTsExtensions": true, "noEmit": true,
    "paths": { "@goscript/*": ["./output/@goscript/*"] }
  },
  "include": ["entry.ts"]
}
`

// buildJS builds the app in the current directory to JavaScript using
// GoScript, bundling it into app.gs.js in the output directory with bun.
func buildJS(c *config.Config) error {
	for _, tool := range []string{"goscript", "bun"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("the js platform requires %s; install goscript with `go install github.com/s4wave/goscript/cmd/goscript@latest` and bun from https://bun.sh", tool)
		}
	}
	tags := "goscript,purego,noasm"
	if c.Build.Debug {
		tags += ",debug"
	}
	pkg, mainFile, err := jsMainPackage(tags)
	if err != nil {
		return err
	}

	work, err := jsWorkDir(pkg)
	if err != nil {
		return err
	}
	output := filepath.Join(work, "output")
	err = os.RemoveAll(output)
	if err != nil {
		return err
	}
	err = exec.Major().Run("goscript", "compile", "-p", ".", "--output", output, "-b", "-tags="+tags)
	if err != nil {
		return fmt.Errorf("error compiling with goscript: %w", err)
	}

	// The dependencies are installed again when the package.json changes,
	// so that a cached work directory gets newly added dependencies.
	pkgJSON := filepath.Join(work, "package.json")
	oldPkgJSON, _ := os.ReadFile(pkgJSON)
	install := string(oldPkgJSON) != jsPackageJSON
	err = os.WriteFile(pkgJSON, []byte(jsPackageJSON), 0666)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(work, "tsconfig.json"), []byte(jsTSConfig), 0666)
	if err != nil {
		return err
	}
	entry := fmt.Sprintf("import { main } from %q\n\nawait main()\n", "@goscript/"+pkg+"/"+mainFile+".gs.ts")
	err = os.WriteFile(filepath.Join(work, "entry.ts"), []byte(entry), 0666)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(work, "node_modules")); err != nil || install {
		err = exec.Major().SetDir(work).Run("bun", "install")
		if err != nil {
			return err
		}
	}

	args := []string{"build", "entry.ts", "--target", "browser", "--format", "esm", "--outfile", filepath.Join(c.Build.Output, "app.gs.js")}
	if !c.Build.Debug {
		// Full minification puts many statements on one line, which is fine
		// for tree.AutoPlanName, since its //goscript:callsite directive makes
		// GoScript pass it the Go file and line of each call.
		args = append(args, "--minify")
	}
	err = exec.Major().SetDir(work).Run("bun", args...)
	if err != nil {
		return fmt.Errorf("error bundling with bun: %w", err)
	}
	return makeFiles(c)
}

// jsMainPackage returns the import path of the main package in the current
// directory for GoScript with the given build tags, and the name of the file
// (without the .go extension) that contains its main function.
func jsMainPackage(tags string) (pkg, mainFile string, err error) {
	out, err := exec.Minor().SetEnv("GOOS", "js").SetEnv("GOARCH", "wasm").
		Output("go", "list", "-tags", tags, "-f", "{{.ImportPath}}{{range .GoFiles}} {{.}}{{end}}", ".")
	if err != nil {
		return "", "", err
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return "", "", errors.New("no Go package found in the current directory")
	}
	fset := token.NewFileSet()
	for _, fn := range fields[1:] {
		f, err := parser.ParseFile(fset, fn, nil, parser.SkipObjectResolution)
		if err != nil {
			return "", "", err
		}
		if f.Name.Name != "main" {
			return "", "", fmt.Errorf("package %s is not a main package", fields[0])
		}
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil && fd.Name.Name == "main" {
				return fields[0], strings.TrimSuffix(fn, ".go"), nil
			}
		}
	}
	return "", "", fmt.Errorf("no main function found in package %s", fields[0])
}

// jsWorkDir returns the directory in the user cache directory that is used
// for compiling and bundling the given package with GoScript, creating it
// if it does not already exist.
func jsWorkDir(pkg string) (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cache, "cogentcore", "goscript", filepath.FromSlash(pkg))
	return dir, os.MkdirAll(dir, 0777)
}
