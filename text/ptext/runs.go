// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ptext

import (
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/text/rich"
	"github.com/go-text/typesetting/shaping"
)

// Runs is a collection of text rendering runs, where each Run
// is the output from a corresponding Span of input text.
// Each input span is defined by a shared set of styling parameters,
// but the corresponding output Run may contain multiple separate
// outputs, due to finer-grained constraints.
type Runs struct {
	// Spans are the input source that generated these output Runs.
	// There should be a 1-to-1 correspondence between each Span and each Run.
	Spans rich.Spans

	// Runs are the list of text rendering runs.
	Runs []Run
}

// Run is a span of output text, corresponding to the span input.
// Each input span is defined by a shared set of styling parameters,
// but the corresponding output Run may contain multiple separate
// sub-spans, due to finer-grained constraints.
type Run struct {
	// Subs contains the sub-spans that together represent the input.
	Subs []shaping.Output

	// Index is our index within the collection of Runs.
	Index int

	// BgPaths are path drawing items for background renders.
	BgPaths render.Render

	// DecoPaths are path drawing items for text decorations.
	DecoPaths render.Render

	// StrikePaths are path drawing items for strikethrough decorations.
	StrikePaths render.Render
}
