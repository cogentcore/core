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
	"cogentcore.org/core/text/rich"
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
	opts.FontFamily = rich.Serif
	opts.Header = HeaderLeftPageNumber("This is a test header")
	buff := bytes.Buffer{}
	PDF(&buff, opts, b)
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
