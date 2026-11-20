// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"testing"
	"time"

	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenHelp tests that the help of the app opens the documentation in
// a window, once, which is raised when it is opened again, with the home
// page, and that [SetAppHelp] sets [core.AppHelp] for the app menu.
func TestOpenHelp(t *testing.T) {
	prev := core.AppHelp
	defer func() { core.AppHelp = prev }()
	SetAppHelp(exampleContent)
	require.NotNil(t, core.AppHelp)
	b := core.NewBody()
	core.AssertRender(t, b, "help", func() {
		helps := func() []*Content {
			var cts []*Content
			for _, rw := range core.AllRenderWindows {
				sc := rw.MainScene()
				sc.AsyncLock()
				sc.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
					if ct, ok := cw.(*Content); ok {
						cts = append(cts, ct)
					}
					return true
				})
				sc.AsyncUnlock()
			}
			return cts
		}
		for range 2 {
			b.AsyncLock()
			core.AppHelp(b)
			b.AsyncUnlock()
			time.Sleep(200 * time.Millisecond)
		}
		cts := helps()
		require.Len(t, cts, 1, "one help window, raised the second time")
		ct := cts[0]
		ct.AsyncLock()
		defer ct.AsyncUnlock()
		assert.True(t, ct.noWebURL)
		assert.Equal(t, "", ct.current.Page.URL, "the home page")
	})
}
