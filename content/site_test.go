// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/content/bcontent"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// htmlOnlyPages are the example pages rendered by [TestContentPages], which
// define the target appearance for the pre-rendered website.
var htmlOnlyPages = []string{"", "button", "widget", "value-binding"}

// TestContentPages renders each of [htmlOnlyPages] in the example content at
// a desktop size, along with a text dump of the left navigation trees, which
// serve as the reference for the pre-rendered website.
func TestContentPages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	for _, url := range htmlOnlyPages {
		name := url
		if name == "" {
			name = "home"
		}
		t.Run(name, func(t *testing.T) {
			b := core.NewBody()
			b.Styler(func(s *styles.Style) {
				s.Min.Set(units.Dp(1280), units.Dp(800))
			})
			ct := NewContent(b).SetSource(exampleContent)
			b.AddTopBar(func(bar *core.Frame) {
				core.NewToolbar(bar).Maker(func(p *tree.Plan) {
					ct.MakeToolbar(p)
				})
			})
			b.AssertRender(t, filepath.Join("pages", name), func() {
				ct.AsyncLock()
				ct.Open(url)
				ct.AsyncUnlock()
			}, func() {
				time.Sleep(200 * time.Millisecond) // to give the image time to load
				dumpLeftTrees(t, ct, filepath.Join("testdata", "pages", name+"-tree.txt"))
			})
		})
	}
}

// dumpLeftTrees writes a text outline of the trees in the left frame,
// with selected and closed state, to the given file.
func dumpLeftTrees(t *testing.T, ct *Content, fname string) {
	var sb strings.Builder
	ct.leftFrame.WalkDown(func(n tree.Node) bool {
		tr, ok := n.(*core.Tree)
		if !ok {
			return tree.Continue
		}
		depth := 0
		for p := tr.AsTree().Parent; p != nil; p = p.AsTree().Parent {
			if _, ok := p.(*core.Tree); ok {
				depth++
			}
		}
		flags := ""
		if tr.StateIs(states.Selected) {
			flags += " [selected]"
		}
		if tr.NumChildren() > 0 && tr.Closed {
			flags += " [closed]"
		}
		fmt.Fprintf(&sb, "%s%s%s\n", strings.Repeat("    ", depth), tr.Text, flags)
		return tree.Continue
	})
	os.MkdirAll(filepath.Dir(fname), 0777)
	if err := os.WriteFile(fname, []byte(sb.String()), 0666); err != nil {
		t.Error(err)
	}
}

func TestGenerateSite(t *testing.T) {
	args := os.Args
	os.Args = []string{args[0], "-generatehtml"}
	defer func() { os.Args = args }()

	b := core.NewBody()
	ct := NewContent(b).SetSource(exampleContent)
	site := &bcontent.Site{}
	assert.NoError(t, jsonx.Read(site, strings.NewReader(generateSite(ct))))

	pages := map[string]*bcontent.PreRenderPage{}
	for _, prp := range site.Pages {
		pages[prp.URL] = prp
	}
	assert.GreaterOrEqual(t, len(pages), 5)
	bt := pages["button"]
	assert.Equal(t, "Properties", bt.TOC[0].Text)
	assert.Equal(t, "properties", bt.TOC[0].ID)
	assert.Equal(t, 5, bt.TOC[5].Level)
	assert.Contains(t, bt.HTML, `<h1 `)
	assert.Contains(t, bt.HTML, `<h2 id="properties"`)
	assert.Contains(t, bt.HTML, `<pre class="code"><code class="language-go">`)
	assert.NotContains(t, bt.HTML, "<textarea") // code only, without editors or results
	assert.Contains(t, pages[""].HTML, "<tr>")
	assert.Empty(t, pages[""].TOC)

	assert.Equal(t, []string{"Concepts", "Widgets", "Other"}, []string{site.Categories[0].Name, site.Categories[1].Name, site.Categories[2].Name})
	assert.Equal(t, "Value binding", site.Categories[0].Pages[0].Name)
	assert.Contains(t, site.CSS, "@media (prefers-color-scheme: dark)")
}
