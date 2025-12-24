// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
)

// RunTest runs a test for given test case.
func RunTest(t *testing.T, nm string, f func() *core.Body) {
	b := f()
	// b.AssertRender(t, "text-only")
	showed := make(chan struct{})
	b.OnFinal(events.Show, func(e events.Event) {
		showed <- struct{}{}
	})
	b.RunWindow()
	<-showed

	opts := NewOptions()
	opts.Header = NoFirst(HeaderLeftPageNumber("This is a test header"))
	opts.Title = CenteredTitle("This is a Profound Statement of Something Important", "Bea A. Author", "University of Twente<br>Department of Physiology", "March 1, 2024", `<a href="https://example.com/testing">https://example.com/testing</a>`, "The thing about this paper is that it is dealing with an issue that should be given more attention, but perhaps it really is hard to understand and that makes it difficult to get the attention it deserves. In any case, we are very proud.")
	buff := bytes.Buffer{}
	b.AsyncLock()
	PDF(&buff, opts, b)
	b.AsyncUnlock()
	os.Mkdir("testdata", 0777)
	os.WriteFile(filepath.Join("testdata", nm)+".pdf", buff.Bytes(), 0666)
}

func TestTextOnly(t *testing.T) {
	ttx := "This is testing text, it is <i>only</i> a test. Do not be <b>alarmed</b>. The text must be at least a certain amount wide so that we can see how it <u>flows</u> up to the margin and judge the typesetting qualities etc."
	RunTest(t, "text-only", func() *core.Body {
		b := core.NewBody()
		b.Styler(func(s *styles.Style) {
			s.Min.X.Ch(80)
		})
		for i := range 200 {
			core.NewText(b).SetText(fmt.Sprintf("Line %d: %s", i, ttx))
		}
		return b
	})
}
