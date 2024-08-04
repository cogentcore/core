// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package web provides functions for building Cogent Core apps for the web.
package web

import (
	"crypto/sha1"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/cmd/core/config"
	"cogentcore.org/core/cmd/core/rendericon"
	"cogentcore.org/core/pages/ppath"
	strip "github.com/grokify/html-strip-tags-go"
)

// Build builds an app for web using the given configuration information.
func Build(c *config.Config) error {
	output := filepath.Join(c.Build.Output, "app.wasm")
	opath := output
	if c.Web.Gzip {
		opath += ".orig"
	}
	err := exec.Major().SetEnv("GOOS", "js").SetEnv("GOARCH", "wasm").Run("go", "build", "-o", opath, "-ldflags", config.LinkerFlags(c))
	if err != nil {
		return err
	}
	if c.Web.Gzip {
		err = exec.RemoveAll(output + ".orig.gz")
		if err != nil {
			return err
		}
		err = exec.Run("gzip", output+".orig")
		if err != nil {
			return err
		}
		err = os.Rename(output+".orig.gz", output)
		if err != nil {
			return err
		}
	}
	return makeFiles(c)
}

// makeFiles makes the necessary static web files based on the given configuration information.
func makeFiles(c *config.Config) error {
	odir := c.Build.Output

	if c.Web.RandomVersion {
		t := time.Now().UTC().String()
		c.Version = fmt.Sprintf(`%x`, sha1.Sum([]byte(t)))
	}

	// The about text may contain HTML, which we need to get rid of.
	// It is trusted, so we do not need a more advanced sanitizer.
	c.About = strip.StripTags(c.About)

	wej := []byte(wasmExecJS)
	err := os.WriteFile(filepath.Join(odir, "wasm_exec.js"), wej, 0666)
	if err != nil {
		return err
	}

	ajs, err := makeAppJS(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "app.js"), ajs, 0666)
	if err != nil {
		return err
	}

	awjs, err := makeAppWorkerJS(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "app-worker.js"), awjs, 0666)
	if err != nil {
		return err
	}

	man, err := makeManifestJSON(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "manifest.webmanifest"), man, 0666)
	if err != nil {
		return err
	}

	acs := []byte(appCSS)
	err = os.WriteFile(filepath.Join(odir, "app.css"), acs, 0666)
	if err != nil {
		return err
	}

	preRenderHTML, err := exec.Output("go", "run", "-tags", "offscreen,generatehtml", ".")
	if err != nil {
		return err
	}
	preRenderHTMLIndex := preRenderHTML
	pagesPreRenderData := &ppath.PreRenderData{}
	if strings.HasPrefix(preRenderHTML, "{") {
		err := jsonx.Read(pagesPreRenderData, strings.NewReader(preRenderHTML))
		if err != nil {
			return err
		}
		preRenderHTMLIndex = pagesPreRenderData.HTML[""]
		if c.Pages == "" {
			c.Pages = "content"
		}
	}
	iht, err := makeIndexHTML(c, "", "", "", preRenderHTMLIndex)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "index.html"), iht, 0666)
	if err != nil {
		return err
	}

	if c.Pages != "" {
		err := makePages(c, pagesPreRenderData)
		if err != nil {
			return err
		}
	}

	err = os.MkdirAll(filepath.Join(odir, "icons"), 0777)
	if err != nil {
		return err
	}
	sizes := []int{32, 192, 512}
	for _, size := range sizes {
		ic, err := rendericon.Render(size)
		if err != nil {
			return err
		}
		err = imagex.Save(ic, filepath.Join(odir, "icons", strconv.Itoa(size)+".png"))
		if err != nil {
			return err
		}
	}
	err = exec.Run("cp", "icon.svg", filepath.Join(odir, "icons", "svg.svg"))
	if err != nil {
		return err
	}

	return nil
}

// makePages makes a directory structure of pages for
// the core pages located at [config.Config.Pages].
func makePages(c *config.Config, preRenderData *ppath.PreRenderData) error {
	return filepath.WalkDir(c.Pages, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		path = strings.ReplaceAll(path, `\`, "/")
		path = strings.TrimSuffix(path, "index.md")
		path = strings.TrimSuffix(path, ".md")
		path = strings.TrimPrefix(path, c.Pages)
		path = strings.TrimPrefix(path, "/")
		path = strings.TrimSuffix(path, "/")
		if ppath.Draft(path) {
			return nil
		}
		path = ppath.Format(path)
		if path == "" { // exclude root index
			return nil
		}
		opath := filepath.Join(c.Build.Output, path)
		err = os.MkdirAll(opath, 0777)
		if err != nil {
			return err
		}
		title := ppath.Label(path, c.Name)
		if title != c.Name {
			title += " • " + c.Name
		}
		b, err := makeIndexHTML(c, ppath.BasePath(path), title, preRenderData.Description[path], preRenderData.HTML[path])
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(opath, "index.html"), b, 0666)
	})
}
