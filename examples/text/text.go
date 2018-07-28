// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/goki/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/ki"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	width := 1024
	height := 768

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	win := gi.NewWindow2D("gogi-text-test", "GoGi Text Test", width, height, true) // true = pixel sizes

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()
	vp.Fill = true

	// style sheet
	var css = ki.Props{
		"kbd": ki.Props{
			"color": "blue",
		},
	}
	vp.CSS = css

	vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	trow := vlay.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutRow
	trow.SetStretchMaxWidth()

	trow.AddNewChild(gi.KiT_Stretch, "str1")
	title := trow.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.Text = `This is a <b>test</b> of the
<span style="color:red">various</span> <i>GoGi</i> Text elements<br>
<large>Shortcuts: <kbd>Ctrl+Alt+P</kbd> = Preferences,
<kbd>Ctrl+Alt+E</kbd> = Editor, <kbd>Ctrl/Cmd +/-</kbd> = zoom</large><br>
Other styles: <u>underlining</u> and <abbr>abbr dotted uline</abbr> and <strike>strikethrough</strike><br>
<q>and</q> <mark>marked text</mark> and <span style="text-decoration:overline">overline</span>
and Sub<sub>script</sub> and Super<sup>script</sup>`
	title.SetProp("text-align", gi.AlignRight)
	title.SetProp("vertical-align", gi.AlignTop)
	title.SetProp("font-family", "Times New Roman, serif")
	title.SetProp("font-size", "x-large")
	// title.SetProp("letter-spacing", 2)
	title.SetProp("line-height", 1.5)

	rtxt := trow.AddNewChild(gi.KiT_Label, "rtxt").(*gi.Label)
	rtxt.Text = "this is to test right margin"

	vlay.AddNewChild(gi.KiT_Space, "spc")

	wrlab := vlay.AddNewChild(gi.KiT_Label, "wrlab").(*gi.Label)
	wrlab.SetProp("word-wrap", true)
	wrlab.SetProp("max-width", "20em")
	wrlab.SetProp("line-height", 1.2)
	wrlab.SetProp("text-indent", "20px")
	wrlab.Text = `Word <u>wrapping</u> should be <span style="color:red">enabled in this label</span> -- this is a test to see if it is.  Usually people use some kind of obscure latin text here -- not really sure why <u>because nobody reads latin anymore,</u> at least nobody I know.  Anyway, let's see how this works.  Also, it should be interesting to determine how word wrapping works with styling -- <large>the styles should properly wrap across the lines</large>.  In addition, there is the question of <b>how built-in breaks interface</b> with the auto-line breaks, and furthermore the question of paragraph breaks versus simple br line breaks.
<p>One major question is the extent to which word wrapping can be made sensitive to the overall size of the containing element -- this is easy when setting a direct fixed width, but word wrapping should also occur as the user resizes the window.</p>
It appears that the <b>end</b> of one paragraph implies the start of a new one, even if you do <i>not</i> insert a <code>p</code> tag.
`

	vlay.AddNewChild(gi.KiT_Stretch, "str")
	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
