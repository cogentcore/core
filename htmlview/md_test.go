// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlview

import (
	"testing"

	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
)

func TestMD(t *testing.T) {
	tests := map[string]string{
		"h1":   `# Test`,
		"h2":   `## Test`,
		"p":    `Test`,
		`code`: "```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n```",
	}
	for nm, s := range tests {
		b := core.NewBody()
		assert.NoError(t, ReadMDString(NewContext(), b, s))
		b.AssertRender(t, "md/"+nm)
	}
}
