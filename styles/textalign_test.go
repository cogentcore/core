// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"testing"

	"cogentcore.org/core/text/text"
	"github.com/stretchr/testify/assert"
)

// TestTextAlignProperties tests the text-align property, as in HTML: its
// initial value is start, not the value of the Start of the layout (as of
// end), and left and right are start and end, in the default direction.
func TestTextAlignProperties(t *testing.T) {
	for val, want := range map[string]text.Aligns{
		"initial": text.Start, "start": text.Start, "end": text.End, "center": text.Center,
		"justify": text.Justify, "left": text.Start, "right": text.End,
	} {
		ts := &Text{}
		ts.Defaults()
		ts.Align = text.Center // not the value, which is set
		ts.FromProperties(nil, map[string]any{"text-align": val}, nil)
		assert.Equal(t, want, ts.Align, val)
	}
}
