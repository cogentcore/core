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

// add adds the sizes of the given file contents to the bundle size.
func (bs *bundleSize) add(b []byte) error {
	bs.Files++
	bs.Size += len(b)
	gz, err := gzipBytes(b, gzip.DefaultCompression)
	if err != nil {
		return err
	}
	bs.Gzip += len(gz)
	if bs.Files == 1 || bs.Brotli > 0 {
		bs.Brotli += brotliSize(b)
	}
	return nil
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

// processBundle measures the size of the app bundle in the build output
// directory (see [bundleFiles]), and gzips its files in place if
// [config.Web.Gzip] is on, in which case [Serve] serves them with
// Content-Encoding gzip, as must any other server of the app.
func processBundle(c *config.Config) (*bundleSize, error) {
	bs := &bundleSize{}
	for _, f := range bundleFiles(c) {
		fn := filepath.Join(c.Build.Output, f)
		b, err := os.ReadFile(fn)
		if err != nil {
			return nil, err
		}
		err = bs.add(b)
		if err != nil {
			return nil, err
		}
		if !c.Web.Gzip {
			continue
		}
		gz, err := gzipBytes(b, gzip.BestCompression)
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(fn, gz, 0666)
		if err != nil {
			return nil, err
		}
	}
	return bs, nil
}

// reportBundle prints the given size of the app bundle.
func reportBundle(c *config.Config, bs *bundleSize) {
	name := "app.wasm"
	if IsJS(c) {
		name = "app.gs.js"
	}
	msg := "bundle " + name + ": " + bs.String()
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
