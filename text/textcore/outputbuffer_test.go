// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"fmt"
	"io"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/lines"
)

// The output that an OutputBuffer monitors in a goroutine is appended to
// its lines, and the editor is updated for it with the render lock, while
// the event loop renders it.
func TestOutputBuffer(t *testing.T) {
	b := core.NewBody()
	tb := lines.NewLines()
	// It is big enough for all of the output, so that it has no scrollbars.
	NewEditor(b).SetLines(tb).Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(15), units.Em(15))
	})
	r, w := io.Pipe()
	ob := &OutputBuffer{Output: r, Lines: tb, Batch: time.Millisecond}
	b.AssertRender(t, "output-buffer", func() {
		done := make(chan struct{})
		go func() {
			ob.MonitorOutput()
			close(done)
		}()
		for i := range 8 {
			fmt.Fprintf(w, "output line %d\n", i)
			time.Sleep(time.Millisecond)
		}
		w.Close()
		<-done
	})
}
