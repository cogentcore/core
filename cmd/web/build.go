// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package web provides functions for building Cogent Core apps for the web.
package web

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/cmd/config"
	"cogentcore.org/core/cmd/rendericon"
	"cogentcore.org/core/content/bcontent"
	strip "github.com/grokify/html-strip-tags-go"
)

// Build builds an app for web using the given configuration information.
func Build(c *config.Config) error {
	if c.Web.HTMLOnly {
		return buildHTMLOnly(c)
	}
	if IsJS(c) {
		return buildJS(c)
	}
	output := filepath.Join(c.Build.Output, "app.wasm")
	opath := output
	if c.Web.Gzip {
		opath += ".orig"
	}
	args := []string{"build", "-o", opath, "-ldflags", config.LinkerFlags(c)}
	if c.Build.Trimpath {
		args = append(args, "-trimpath")
	}
	err := exec.Major().SetEnv("GOOS", "js").SetEnv("GOARCH", "wasm").Run("go", args...)
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

	if !IsJS(c) {
		err := os.WriteFile(filepath.Join(odir, "wasm_exec.js"), []byte(wasmExecJS), 0666)
		if err != nil {
			return err
		}
	}

	ajs, err := makeAppJS(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "app.js"), ajs, 0666)
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

	preRenderHTML := ""
	if c.Web.GenerateHTML {
		preRenderHTML, err = generateHTML()
		if err != nil {
			return err
		}
	}
	// A custom index page replaces the pre-rendered preview of the root page
	// only; the pre-rendered previews of any content pages are still used.
	indexPage, err := readIndexPage()
	if err != nil {
		return err
	}
	site, err := readSite(preRenderHTML)
	if err != nil {
		return err
	}
	awjs, err := makeAppWorkerJS(c, site != nil)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "app-worker.js"), awjs, 0666)
	if err != nil {
		return err
	}
	if site != nil {
		err = makeContentSite(c, site, true, indexPage)
		if err != nil {
			return err
		}
		return makeIcons(c)
	}

	prindex := &bcontent.PreRenderPage{
		Page: bcontent.Page{Name: c.Name},
		HTML: preRenderHTML,
	}
	if c.About != "" {
		prindex.Description = c.About
	}
	iht, err := makeIndexHTML(c, "", prindex, indexPage)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "index.html"), iht, 0666)
	if err != nil {
		return err
	}

	// The 404 page is just the same as the index page, with an updated base path.
	// The logic in the home page can then handle the error appropriately.
	notFound, err := makeIndexHTML(c, notFoundBasePath(c), prindex, "")
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "404.html"), notFound, 0666)
	if err != nil {
		return err
	}

	return makeIcons(c)
}

// makeIcons makes the app icon files in the icons directory of the output.
func makeIcons(c *config.Config) error {
	odir := c.Build.Output
	err := os.MkdirAll(filepath.Join(odir, "icons"), 0777)
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

// notFoundBasePath returns the base path for the 404.html page, which is
// for pages one level below the root.
func notFoundBasePath(c *config.Config) string {
	// TODO: this is a temporary hack to fix the 404 page for multi-nested old URLs in the Cogent Core Docs.
	if c.Name == "Cogent Core Docs" {
		if c.Build.Trimpath {
			return "https://www.cogentcore.org/core/" // production
		}
		return "http://localhost:8080/" // dev
	}
	return "../"
}
