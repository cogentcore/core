// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

//go:generate go run gen/scripts.go

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cogentcore.org/core/cmd/core/config"
	"cogentcore.org/core/cmd/core/rendericon"
	"cogentcore.org/core/grows/images"
	"cogentcore.org/core/xe"
)

// Build builds an app for web using the given configuration information.
func Build(c *config.Config) error {
	output := filepath.Join("bin", "web", "app.wasm")
	opath := output
	if c.Web.Gzip {
		opath += ".orig"
	}
	err := xe.Major().SetEnv("GOOS", "js").SetEnv("GOARCH", "wasm").Run("go", "build", "-o", opath, "-ldflags", config.VersionLinkerFlags())
	if err != nil {
		return err
	}
	if c.Web.Gzip {
		err = xe.RemoveAll(output + ".orig.gz")
		if err != nil {
			return err
		}
		err = xe.Run("gzip", output+".orig")
		if err != nil {
			return err
		}
		err = os.Rename(output+".orig.gz", output)
		if err != nil {
			return err
		}
	}
	return MakeFiles(c)
}

// MakeFiles makes the necessary static web files based on the given configuration information.
func MakeFiles(c *config.Config) error {
	odir := filepath.Join("bin", "web")

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

	err = os.MkdirAll(filepath.Join(odir, "icons"), 0777)
	if err != nil {
		return err
	}
	ic192, err := rendericon.Render(192)
	if err != nil {
		return err
	}
	err = images.Save(ic192, filepath.Join(odir, "icons", "192.png"))
	if err != nil {
		return err
	}
	ic512, err := rendericon.Render(512)
	if err != nil {
		return err
	}
	err = images.Save(ic512, filepath.Join(odir, "icons", "512.png"))
	if err != nil {
		return err
	}
	err = xe.Run("cp", "icon.svg", filepath.Join(odir, "icons", "svg.svg"))
	if err != nil {
		return err
	}

	return nil
}
