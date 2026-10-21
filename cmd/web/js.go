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
	"io/fs"
	"maps"
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
	// GoScript and bun are installed automatically if they are not found.
	err := ensureJSTools()
	if err != nil {
		return err
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
	// Dead code elimination leaves the functions and methods that the
	// program can never call out of the bundle (unless the environment
	// already sets GOSCRIPT_ELIMINATE_DEAD_CODE, such as to 0 to turn it off).
	// Versions of goscript without it ignore the variable.
	gs := exec.Major()
	gs.Env = maps.Clone(gs.Env)
	if gs.Env == nil {
		gs.Env = map[string]string{}
	}
	if _, ok := os.LookupEnv("GOSCRIPT_ELIMINATE_DEAD_CODE"); !ok {
		gs.Env["GOSCRIPT_ELIMINATE_DEAD_CODE"] = "1"
	}
	for k, v := range jsExternalEmbedEnv() {
		if _, ok := os.LookupEnv(k); !ok {
			gs.Env[k] = v
		}
	}
	gsArgs := []string{"compile", "-p", ".", "--output", output, "-b", "-tags=" + tags}
	for _, f := range c.Web.DeferredFunctions {
		gsArgs = append(gsArgs, "--deferred-function", f)
	}
	err = gs.Run("goscript", gsArgs...)
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

	args := []string{"build", "entry.ts", "--target", "browser", "--format", "esm"}
	err = removeJSChunks(c.Build.Output)
	if err != nil {
		return err
	}
	if len(c.Web.DeferredFunctions) > 0 {
		// The deferred functions are dynamic imports, each in its own chunk.
		args = append(args, "--splitting", "--outdir", c.Build.Output, "--entry-naming", "app.gs.js", "--chunk-naming", jsChunkPrefix+"[hash].js")
	} else {
		args = append(args, "--outfile", filepath.Join(c.Build.Output, "app.gs.js"))
	}
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
	err = copyExternalEmbeds(output, c.Build.Output)
	if err != nil {
		return err
	}
	return makeFiles(c)
}

// jsChunkPrefix is the prefix of the names of the files of the bundle other
// than app.gs.js: the chunks of [config.Web.DeferredFunctions].
const jsChunkPrefix = "app.gs.chunk-"

// removeJSChunks removes the chunk files of an earlier build from the given
// build output directory.
func removeJSChunks(odir string) error {
	for _, f := range jsChunkFiles(odir) {
		err := os.Remove(filepath.Join(odir, f))
		if err != nil {
			return err
		}
	}
	return nil
}

// jsChunkFiles returns the names of the chunk files of the bundle in the
// given build output directory (see [jsChunkPrefix]).
func jsChunkFiles(odir string) []string {
	entries, _ := os.ReadDir(odir)
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), jsChunkPrefix) && strings.HasSuffix(e.Name(), ".js") {
			files = append(files, e.Name())
		}
	}
	return files
}

// jsExternalEmbedPatterns are the base name patterns of the embedded files of
// embed.FS variables that GoScript writes as separate files, which the app
// fetches when it first opens them, instead of into the bundle: media files,
// such as the images of a content site, which are only needed when a page
// shows them, and which do not compress well, so that they would make the
// download of the bundle much larger.
var jsExternalEmbedPatterns = []string{
	"*.png", "*.jpg", "*.jpeg", "*.gif", "*.webp", "*.avif", "*.bmp", "*.ico", "*.svg",
	"*.mp3", "*.wav", "*.ogg", "*.mp4", "*.webm", "*.pdf",
}

// jsExternalEmbedMinSize is the size from which a file that matches
// [jsExternalEmbedPatterns] is external; smaller files are not worth a request.
const jsExternalEmbedMinSize = 8 << 10

// jsExternalEmbedDir is the directory of the external embedded files, which
// GoScript writes beside the output of each package and the app loads from
// beside app.gs.js, so they are all copied into it in the output directory.
const jsExternalEmbedDir = "@embed"

// jsExternalEmbedEnv returns the GoScript environment variables that select
// the external embedded files (see [jsExternalEmbedPatterns]), unless the
// environment already sets them. Versions of goscript without them ignore
// them, and then inline all files as before.
func jsExternalEmbedEnv() map[string]string {
	return map[string]string{
		"GOSCRIPT_EXTERNAL_EMBED":          strings.Join(jsExternalEmbedPatterns, ","),
		"GOSCRIPT_EXTERNAL_EMBED_MIN_SIZE": fmt.Sprint(jsExternalEmbedMinSize),
	}
}

// copyExternalEmbeds copies the external embedded files in the GoScript output
// directory (see [jsExternalEmbedDir]) into the given build output directory.
func copyExternalEmbeds(output, odir string) error {
	dst := filepath.Join(odir, jsExternalEmbedDir)
	err := os.RemoveAll(dst)
	if err != nil {
		return err
	}
	return filepath.WalkDir(output, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(filepath.Dir(p)) != jsExternalEmbedDir {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		err = os.MkdirAll(dst, 0777)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, d.Name()), b, 0666)
	})
}

// externalEmbedFiles returns the paths of the external embedded files in the
// given build output directory (see [jsExternalEmbedDir]), relative to it.
func externalEmbedFiles(odir string) []string {
	entries, _ := os.ReadDir(filepath.Join(odir, jsExternalEmbedDir))
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, jsExternalEmbedDir+"/"+e.Name())
		}
	}
	return files
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
// if it does not already exist. The CORE_JS_WORK_DIR environment variable
// sets another directory, such as for builds of the same package from
// several worktrees.
func jsWorkDir(pkg string) (string, error) {
	if dir := os.Getenv("CORE_JS_WORK_DIR"); dir != "" {
		return dir, os.MkdirAll(dir, 0777)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cache, "cogentcore", "goscript", filepath.FromSlash(pkg))
	return dir, os.MkdirAll(dir, 0777)
}
