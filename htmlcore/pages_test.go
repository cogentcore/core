// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
)

// TestPages is the regression test of the rendering of whole pages: the
// pages of the fidelity test (testdata/fidelity/*.html) and the snapshots of
// real pages (testdata/snapshots, see [TestSnapshot]), which are all local
// files, are rendered and compared with the images in testdata/pages, as
// for the other render tests. When a change of the rendering is checked
// and is right, the images are updated with:
//
//	go test -run TestPages -tags update
//
// The images are of at most the top pagesMaxHeight dots of each page.
// For a page that fails, the expected image and the new one are side by
// side in testdata/pages/.../name.compare.png.
// Run with -run TestPages/name to test one page.
func TestPages(t *testing.T) {
	type page struct{ name, file string }
	var pages []page
	fid, _ := filepath.Glob("testdata/fidelity/*.html")
	for _, f := range fid {
		pages = append(pages, page{"fidelity/" + strings.TrimSuffix(filepath.Base(f), ".html"), f})
	}
	snaps, _ := filepath.Glob(filepath.Join(snapshotsDir, "*", "index.html"))
	for _, f := range snaps {
		pages = append(pages, page{"snapshots/" + filepath.Base(filepath.Dir(f)), f})
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].name < pages[j].name })
	if len(snaps) == 0 {
		t.Logf("no snapshots in %s: save them with -run TestSnapshot -snapshot", snapshotsDir)
	}

	for _, pg := range pages {
		t.Run(pg.name, func(t *testing.T) {
			width := pagesSnapshotWidth
			if strings.HasPrefix(pg.name, "fidelity/") {
				width = fidelityWidth()
			}
			src, err := os.ReadFile(pg.file)
			if err != nil {
				t.Fatal(err)
			}
			dir, _ := filepath.Abs(filepath.Dir(pg.file))
			remote := map[string]bool{}
			_, img, _ := coreRenderWidth(string(src), dir, "", width, 600, func(u string) { remote[u] = true })
			if img == nil {
				t.Fatal("no image")
			}
			for u := range remote {
				t.Errorf("the page requests a resource on the web: %s", u)
			}
			if r := img.Bounds(); r.Dy() > pagesMaxHeight {
				img = img.SubImage(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+pagesMaxHeight)).(*image.RGBA)
			}
			imagex.Assert(t, img, filepath.Join("pages", pg.name))
			// a failure has the expected image and the new one side by side
			base := filepath.Join("testdata", "pages", pg.name)
			compare := base + ".compare.png"
			os.Remove(compare)
			if _, err := os.Stat(base + ".fail.png"); err == nil {
				savePNG(compare, sideBySide(readPNG(base+".png"), img))
			}
		})
	}
}

const (
	// pagesSnapshotWidth is the width of the viewport of the snapshots
	// in [TestPages], in dp (CSS px).
	pagesSnapshotWidth = 1024

	// pagesMaxHeight is the largest height of the images of [TestPages].
	pagesMaxHeight = 4000
)
