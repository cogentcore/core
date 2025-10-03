// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"bytes"
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

	buff := bytes.Buffer{}
	PDF(&buff, NewOptions(), b)
	os.Mkdir("testdata", 0777)
	os.WriteFile(filepath.Join("testdata", nm)+".pdf", buff.Bytes(), 0666)
}

func TestTextOnly(t *testing.T) {
	ttx := "This is testing text, it is only a test. Do not be alarmed"
	RunTest(t, "text-only", func() *core.Body {
		b := core.NewBody()
		b.Styler(func(s *styles.Style) {
			s.Min.X.Ch(80)
		})
		for range 200 {
			core.NewText(b).SetText(ttx)
		}
		return b
	})
}
