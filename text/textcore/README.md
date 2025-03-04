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

* js text measuring

* textcore base horizontal scrolling and wrap long no-space lines
* prompt on quitting modified file hangs: can't figure it out: dialog is called but never opens, then it hangs

* filetree 3x update and slow repo open

* shaped does not process `\n` https://github.com/go-text/typesetting/issues/185 

* svg opacity processing

* better job finding path fragments from file links -- iteratively try stuff.

* can get zombie nil open buffers in code -- under what circumstances?? autosave renames?

* renderx/images needs transform updates?

### Lowpri

* cleanup unused base stuff

* textfield NewLayout causes dreaded shadow accumulation on popup dialogs. mostly fixed by getting the initial layout size correct, but when it wraps, it will cause this.

* text render highlight region fill in blanks better: hard b/c at run level, doesn't have context.

* xyz physics GrabEyeImg causes crashing with goroutine renderAsync in renderwindow, but otherwise is ok

* optimized next level up render: no clear advantage; not clear what the point is?

