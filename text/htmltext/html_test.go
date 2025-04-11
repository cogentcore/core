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

	trg := `[]: "The "
[italic]: "lazy"
[]: " fox typed in some "
[1.50x bold]: "familiar"
[]: " text"
`
	// fmt.Println(tx.String())
	assert.Equal(t, trg, tx.String())
}

func TestLink(t *testing.T) {
	src := `The <a href="https://example.com">link</a> and`
	tx, err := HTMLToRich([]byte(src), rich.NewStyle(), nil)
	assert.NoError(t, err)

	trg := `[]: "The "
[link [https://example.com] underline fill-color]: "link"
[{End Special}]: ""
[]: " and"
`
	// fmt.Println(tx.String())
	// tx.DebugDump()

	assert.Equal(t, trg, tx.String())

	txt := tx.Join()
	assert.Equal(t, "The link and", string(txt))
}

func TestLinkFmt(t *testing.T) {
	src := `The <a href="https://example.com">link <b>and <i>it</i> is cool</b></a> and`
	tx, err := HTMLToRich([]byte(src), rich.NewStyle(), nil)
	assert.NoError(t, err)

	trg := `[]: "The "
[link [https://example.com] underline fill-color]: "link "
[bold underline fill-color]: "and "
[italic bold underline fill-color]: "it"
[bold underline fill-color]: " is cool"
[underline fill-color]: ""
[{End Special}]: ""
[]: " and"
`
	assert.Equal(t, trg, tx.String())

	os := "The link and it is cool and"
	assert.Equal(t, []rune(os), tx.Join())
}

func TestDemo(t *testing.T) {
	src := `A <b>demonstration</b> of the <i>various</i> features of the <a href="https://cogentcore.org/core">Cogent Core</a> 2D and 3D Go GUI <u>framework</u>`
	tx, err := HTMLToRich([]byte(src), rich.NewStyle(), nil)
	assert.NoError(t, err)

	trg := `[]: "A "
[bold]: "demonstration"
[]: " of the "
[italic]: "various"
[]: " features of the "
[link [https://cogentcore.org/core] underline fill-color]: "Cogent Core"
[{End Special}]: ""
[]: " 2D and 3D Go GUI "
[underline]: "framework"
[]: ""
`

	assert.Equal(t, trg, tx.String())
}

func TestBreak(t *testing.T) {
	src := "Text2D can put <b>HTML</b> <br>formatted<br>Text anywhere you might <i>want</i>"
	tx, err := HTMLToRich([]byte(src), rich.NewStyle(), nil)
	assert.NoError(t, err)

	trg := `[]: "Text2D can put "
[bold]: "HTML"
[]: " "
[]: "
"
[]: "formatted"
[]: "
"
[]: "Text anywhere you might "
[italic]: "want"
[]: ""
`
	assert.Equal(t, trg, tx.String())
}
