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

* comment italics code cursor position is off by 1 to far right.
* dreaded shadow accumulation on popup dialogs. very subtle. e.g. commit dialog -- only input not the ctrl+m chooser.
* shaped random crash: protect with mutex
* better job finding path fragments from file links.
* within line tab rendering
* xyz text rendering
* svg text rendering, markers, lab plot text rotation
* base horizontal scrolling
* cleanup unused base stuff



