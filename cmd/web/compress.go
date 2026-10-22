// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"cogentcore.org/core/base/datasize"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/cmd/config"
)

// bundleFiles returns the names of the files of the app bundle in the
// build output directory: app.wasm for wasm, and app.gs.js and its chunks
// (see [jsChunkPrefix]) for js.
func bundleFiles(c *config.Config) []string {
	if !IsJS(c) {
		return []string{"app.wasm"}
	}
	return append([]string{"app.gs.js"}, jsChunkFiles(c.Build.Output)...)
}

// bundleSize is the total size of the files of an app bundle,
// uncompressed and compressed.
type bundleSize struct {

	// Files is the number of files.
	Files int

	// Size is the uncompressed size in bytes.
	Size int

	// Gzip is the size in bytes with gzip at the default level, which
	// is what web servers that compress responses typically use.
	Gzip int

	// Brotli is the size in bytes with brotli at quality 5, a typical level
	// for compressing responses on the fly, or 0 if the brotli command
	// is not available.
	Brotli int
}

// fileSize returns the size of a bundle of one file with the given contents.
func fileSize(b []byte) (*bundleSize, error) {
	gz, err := gzipBytes(b, gzip.DefaultCompression)
	if err != nil {
		return nil, err
	}
	return &bundleSize{Files: 1, Size: len(b), Gzip: len(gz), Brotli: brotliSize(b)}, nil
}

// add adds the given size of other files to the bundle size.
func (bs *bundleSize) add(o *bundleSize) {
	bs.Files += o.Files
	bs.Size += o.Size
	bs.Gzip += o.Gzip
	bs.Brotli += o.Brotli
}

// String returns the sizes in the form used in the build report.
func (bs *bundleSize) String() string {
	s := fmt.Sprintf("%s (gzip %s", datasize.Size(bs.Size), datasize.Size(bs.Gzip))
	if bs.Brotli > 0 {
		s += fmt.Sprintf(", brotli %s", datasize.Size(bs.Brotli))
	}
	s += ")"
	if bs.Files > 1 {
		s += fmt.Sprintf(" in %d files", bs.Files)
	}
	return s
}

// jsStaticImport matches a static import of a file of the bundle in
// the JavaScript output of esbuild, such as from"./app.gs.chunk-x.js",
// but not a dynamic import("./app.gs.chunk-x.js").
var jsStaticImport = regexp.MustCompile(`\b(?:from|import)\s*["']\./(app\.gs(?:\.chunk-[\w-]+)?\.js)["']`)

// startupFiles returns the files of the bundle with the given contents that
// the browser loads at startup: the first one, which is the entry, and the
// files that it imports statically, directly or through each other. The
// other files are chunks of [config.Web.DeferredFunctions], which are only
// loaded when they are first called.
func startupFiles(files []string, contents map[string][]byte) map[string]bool {
	startup := map[string]bool{files[0]: true}
	todo := []string{files[0]}
	for len(todo) > 0 {
		f := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		for _, m := range jsStaticImport.FindAllSubmatch(contents[f], -1) {
			imp := string(m[1])
			if _, ok := contents[imp]; ok && !startup[imp] {
				startup[imp] = true
				todo = append(todo, imp)
			}
		}
	}
	return startup
}

// processBundle measures the size of the app bundle in the build output
// directory (see [bundleFiles]), in total and of the files loaded at
// startup (see [startupFiles]), and gzips its files in place if
// [config.Web.Gzip] is on, in which case [Serve] serves them with
// Content-Encoding gzip, as must any other server of the app.
func processBundle(c *config.Config) (total, startup *bundleSize, err error) {
	files := bundleFiles(c)
	contents := map[string][]byte{}
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(c.Build.Output, f))
		if err != nil {
			return nil, nil, err
		}
		contents[f] = b
	}
	atStartup := startupFiles(files, contents)
	total, startup = &bundleSize{}, &bundleSize{}
	for _, f := range files {
		fn := filepath.Join(c.Build.Output, f)
		b := contents[f]
		fs, err := fileSize(b)
		if err != nil {
			return nil, nil, err
		}
		total.add(fs)
		if atStartup[f] {
			startup.add(fs)
		}
		if !c.Web.Gzip {
			continue
		}
		gz, err := gzipBytes(b, gzip.BestCompression)
		if err != nil {
			return nil, nil, err
		}
		err = os.WriteFile(fn, gz, 0666)
		if err != nil {
			return nil, nil, err
		}
	}
	return total, startup, nil
}

// reportBundle prints the given total size of the app bundle, and the
// size of its files loaded at startup if that is not all of them.
func reportBundle(c *config.Config, total, startup *bundleSize) {
	name := "app.wasm"
	if IsJS(c) {
		name = "app.gs.js"
	}
	msg := "bundle " + name + ": " + total.String()
	if startup.Files < total.Files {
		msg += "; loaded at startup: " + startup.String()
	}
	if c.Web.Gzip {
		msg += "; gzipped in place (serve with Content-Encoding: gzip)"
	}
	logx.PrintlnWarn(msg)
}

// gzipBytes returns the given data compressed with gzip at the given level.
func gzipBytes(b []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	_, err = gw.Write(b)
	if err != nil {
		return nil, err
	}
	err = gw.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// brotliSize returns the size of the given data compressed with brotli
// at quality 5 by the brotli command, or 0 if it is not available.
// (Quality 11, which hosts use for files that they compress ahead of time,
// is much slower, such as 100 seconds for a 60 MB wasm file.)
func brotliSize(b []byte) int {
	path, err := osexec.LookPath("brotli")
	if err != nil {
		return 0
	}
	cmd := osexec.Command(path, "-c", "-q", "5")
	cmd.Stdin = bytes.NewReader(b)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	return len(out)
}

// acceptsGzip returns whether the given Accept-Encoding header value
// accepts gzip.
func acceptsGzip(accept string) bool {
	for _, e := range strings.Split(accept, ",") {
		name, params, _ := strings.Cut(e, ";")
		name = strings.TrimSpace(name)
		if name != "gzip" && name != "*" {
			continue
		}
		q, ok := strings.CutPrefix(strings.ReplaceAll(params, " ", ""), "q=")
		if !ok {
			return true
		}
		v, err := strconv.ParseFloat(q, 64)
		return err == nil && v > 0
	}
	return false
}
