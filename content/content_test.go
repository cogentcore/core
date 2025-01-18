// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"embed"
	"testing"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
)

//go:embed examples/basic/content
var exampleContentParent embed.FS

var exampleContent = fsx.Sub(exampleContentParent, "examples/basic/content")

func TestContentSetSource(t *testing.T) {
	b := core.NewBody()
	ct := NewContent(b).SetSource(exampleContent)
	_ = ct
	b.AssertRender(t, "set-source")
}
