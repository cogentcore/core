// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

// genstdsrc writes stdsrc.zip, the sources of the generic standard library
// packages that interpreted code can import (see stdsrc.go), from the std
// module archive of the mvm module that this module requires.
package main

import (
	"archive/zip"
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// packages are the packages whose sources stdsrc.zip has: generic packages,
// which cannot be bound as values, and the packages that they import other
// than math/bits and unsafe, which are bound.
var packages = []string{"cmp", "iter", "maps", "slices"}

func main() {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/cogentcore/mvm").Output()
	if err != nil {
		log.Fatal(err)
	}
	src, err := zip.OpenReader(filepath.Join(strings.TrimSpace(string(out)), "stdlib", "src.zip"))
	if err != nil {
		log.Fatal(err)
	}
	defer src.Close()

	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, f := range src.File {
		// The entries are <module>@<version>/<package>/<file>.
		_, rest, _ := strings.Cut(f.Name, "@")
		_, name, ok := strings.Cut(rest, "/")
		dir, file := path.Split(name)
		if !ok || !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") || !slices.Contains(packages, strings.TrimSuffix(dir, "/")) {
			continue
		}
		r, err := f.Open()
		if err != nil {
			log.Fatal(err)
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			log.Fatal(err)
		}
		fw, err := w.Create(name)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := fw.Write(data); err != nil {
			log.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("stdsrc.zip", b.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
}
