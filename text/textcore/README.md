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

* docs weird bug issue.
* shaped crashing with <br> html text
* textcore base horizontal scrolling and wrap long no-space lines
* prompt on quitting modified file hangs: can't figure it out: dialog is called but never opens, then it hangs

* undo bug is due to diff updating not tracking undo state? seems to happen after significant diff revert update.

* diff next doesn't scroll both

* renderx/images needs transform updates?
* svg opacity processing

* optimized next level up render

* dotted underline for misspelling
* better job finding path fragments from file links -- iteratively try stuff.
* core/values.go/FontButton -- need font list.
* cleanup unused base stuff
* dreaded shadow accumulation on popup dialogs. very subtle. e.g. commit dialog -- only input not the ctrl+m chooser.
* text render highlight region fill in blanks better: hard b/c at run level, doesn't have context.

* xyz physics GrabEyeImg causes crashing with goroutine renderAsync in renderwindow, but otherwise is ok


