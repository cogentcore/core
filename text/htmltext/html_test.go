// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmltext

import (
	"testing"

	"cogentcore.org/core/text/rich"
	"github.com/stretchr/testify/assert"
)

func TestHTML(t *testing.T) {
	src := `The <i>lazy</i> fox typed in some <span style="font-size:x-large;font-weight:bold">familiar</span> text`
	tx, err := HTMLToRich([]byte(src), rich.NewStyle(), nil)
	assert.NoError(t, err)

	trg := `[]: The 
[italic]: lazy
[]:  fox typed in some 
[1.50x bold]: familiar
[]:  text
`
	assert.Equal(t, trg, tx.String())
}
