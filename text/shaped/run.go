// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"github.com/go-text/typesetting/shaping"
)

// Run is a span of output text, corresponding to an individual [rich]
// Span input. Each input span is defined by a shared set of styling
// parameters, but the corresponding output Run may contain multiple
// separate sub-spans, due to finer-grained constraints.
type Run struct {
	// Subs contains the sub-spans that together represent the input Span.
	Subs []shaping.Output

	// FontSize is the target font size, needed for scaling during render.
	FontSize float32
}
