# textcore: core GUI widgets for text

Textcore has various text-based `core.Widget`s, including:
* `Base` is a base implementation that views `lines.Lines` text content.
* `Editor` is a full-featured text editor built on Base.
* `DiffEditor` provides side-by-side Editors showing the diffs between files.
* `TwinEditors` provides two side-by-side Editors that sync their scrolling.
* `Terminal` provides a VT100-style terminal built on Base (TODO).

A critical design feature is that the Base widget can switch efficiently among different `lines.Lines` content. For example, in Cogent Code there are 2 editor widgets that are reused for all files, including viewing the same file across both editors. Thus, all of the state comes from the underlying Lines buffers.

The `Lines` handles all layout and markup styling, so the Base just renders the results of that. Thus, there is no need for the Editor to ever drive a NeedsLayout call itself: everything is handled in the Render step, including the presence or absence of the scrollbar, which is a little bit complicated because adding a scrollbar changes the effective width and thus the internal layout.

## Files

The underlying `lines.Lines` object does not have any core dependencies, and is designed to manage lines in memory. See [files.go](files.go) for standard functions to provide GUI-based interactions for prompting when a file has been modified on a disk, and prompts for saving a file. These functions can be used on a Lines without any specific widget.

## TODO

* move shaper to renderwindow so popup menus etc don't need to make their own? SVG too!? is every icon getting a shaper?

* svg marker glitch is last remaining bug: debugit!

* check for negative advance and highlighting issues / tests

* emoji, svg, bitmap font rendering: could not get color emoji to work

* SVG, PDF backends

* textcore base test horizontal scrolling

* prompt on quitting modified file hangs: can't figure it out: dialog is called but never opens, then it hangs

* shaped does not process `\n` https://github.com/go-text/typesetting/issues/185 

* better job finding path fragments from file links -- iteratively try stuff.

* renderx/images needs transform updates?

### Lowpri

* code newFiles AddToVCS should default on -- not working

* code: Markup colors are baked in when output is generated -- no remarkup possible!

* TestMarkupSpellErr: still some rich tag format issues but mostly working.. why is this so hard!?

* cleanup unused base stuff

* textfield NewLayout causes dreaded shadow accumulation on popup dialogs. mostly fixed by getting the initial layout size correct, but when it wraps, it will cause this.

* text render highlight region fill in blanks better: hard b/c at run level, doesn't have context.

* xyz physics GrabEyeImg causes crashing with goroutine renderAsync in renderwindow, but otherwise is ok

* optimized next level up render: no clear advantage; not clear what the point is?


* selection crash after undo:

panic: invalid character position for pos, len: 30: pos: 217:37 [recovered]
	panic: invalid character position for pos, len: 30: pos: 217:37
cogentcore.org/core/text/lines.(*Lines).mustValidPos(0x140b56f7798?, {0xd8, 0x25})
	/Users/oreilly/go/src/cogentcore.org/core/text/lines/lines.go:369 +0xd4
cogentcore.org/core/text/lines.(*Lines).region(0x1407a536008, {0xd6, 0x0}, {0xd8, 0x25})
	/Users/oreilly/go/src/cogentcore.org/core/text/lines/lines.go:383 +0xa8
cogentcore.org/core/text/lines.(*Lines).deleteTextImpl(0x1407a536008, {0x101282fec?, 0x14001d38240?}, {0x38?, 0x140b56f7978?})
	/Users/oreilly/go/src/cogentcore.org/core/text/lines/lines.go:474 +0x34
cogentcore.org/core/text/lines.(*Lines).deleteText(0x1407a536008, {0x100d36bd0?, 0x140b56f7a88?}, {0x1012b2548?, 0x0?})
	/Users/oreilly/go/src/cogentcore.org/core/text/lines/lines.go:468 +0x20
cogentcore.org/core/text/lines.(*Lines).DeleteText(0x1407a536008, {0x0?, 0x0?}, {0x0?, 0x0?})
	/Users/oreilly/go/src/cogentcore.org/core/text/lines/api.go:450 +0xb4
cogentcore.org/core/text/textcore.(*Base).deleteSelection(0x14000d69108)
	/Users/oreilly/go/src/cogentcore.org/core/text/textcore/select.go:248 +0x38
cogentcore.org/core/text/textcore.(*Base).InsertAtCursor(0x14000d69108, {0x140b56f7ac0, 0x1, 0x0?})
	/Users/oreilly/go/src/cogentcore.org/core/text/textcore/select.go:284 +0x4c
cogentcore.org/core/text/textcore.(*Editor).keyInputInsertRune(0x14000d69108, {0x102e69e00, 0x140f902bdc0})
	/Users/oreilly/go/src/cogentcore.org/core/text/textcore/editor.go:576 +0x288

