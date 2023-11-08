// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"crypto/sha1"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"goki.dev/goki/config"
	"goki.dev/grog"
	"goki.dev/xe"
)

// Build builds an app for web using the given configuration information.
func Build(c *config.Config) error {
	err := xe.Major().SetEnv("GOOS", "js").SetEnv("GOARCH", "wasm").Run("go", "build", "-o", c.Build.Output+".orig", c.Build.Package)
	if err != nil {
		return err
	}
	err = xe.RemoveAll(c.Build.Output + ".orig.gz")
	if err != nil {
		return err
	}
	err = xe.Run("gzip", c.Build.Output+".orig")
	if err != nil {
		return err
	}
	err = os.Rename(c.Build.Output+".orig.gz", c.Build.Output)
	if err != nil {
		return err
	}
	return MakeFiles(c)
}

// MakeFiles makes the necessary static web files based on the given configuration information.
func MakeFiles(c *config.Config) error {
	odir := filepath.Dir(c.Build.Output)

	if c.Web.RandomVersion {
		t := time.Now().UTC().String()
		c.Version += "-" + fmt.Sprintf(`%x`, sha1.Sum([]byte(t)))
	}

	wej := []byte(WASMExecJS())
	err := os.WriteFile(filepath.Join(odir, "wasm_exec.js"), wej, 0666)
	if err != nil {
		return err
	}

	ajs, err := MakeAppJS(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "app.js"), ajs, 0666)
	if err != nil {
		return err
	}

	awjs, err := MakeAppWorkerJS(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "app-worker.js"), awjs, 0666)
	if err != nil {
		return err
	}

	man, err := MakeManifestJSON(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "manifest.webmanifest"), man, 0666)
	if err != nil {
		return err
	}

	acs := []byte(AppCSS)
	err = os.WriteFile(filepath.Join(odir, "app.css"), acs, 0666)
	if err != nil {
		return err
	}

	iht, err := MakeIndexHTML(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "index.html"), iht, 0666)
	if err != nil {
		return err
	}

	ics := filepath.Join(c.Build.Package, ".goki", "icons")
	err = xe.Run("cp", "-r", ics, odir)
	grog.InitColor()
	if err != nil {
		// an error copying icons is unfortunate but shouldn't sink the whole build
		// for example, building without icons should at least be possible
		slog.Error("error copying icons", "from", ics, "to", odir, "err", err)
	}

	return nil
}
