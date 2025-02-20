# textcore

Textcore has various text-based `core.Widget`s, including:
* `Base` is a base implementation that views `lines.Lines` text content.
* `Editor` is a full-featured text editor built on Base.
* `DiffEditor` provides side-by-side Editors showing the diffs between files.
* `TwinEditors` provides two side-by-side Editors that sync their scrolling.
* `Terminal` provides a VT100-style terminal built on Base (TODO).

A critical design feature is that the Base widget can switch efficiently among different `lines.Lines` content. For example, in Cogent Code there are 2 editor widgets that are reused for all files, including viewing the same file across both editors. Thus, all of the state comes from the underlying Lines buffers.

The `Lines` handles all layout and markup styling, so the Base just renders the results of that. Thus, there is no need for the Editor to ever drive a NeedsLayout call itself: everything is handled in the Render step, including the presence or absence of the scrollbar, which is a little bit complicated because adding a scrollbar changes the effective width and thus the internal layout.

# TODO

* autoscroll too sensitive / biased
* dynamic scroll re-layout
* base horizontal scrolling
* link handling
* cleanup unused base stuff
* outputbuffer formatting
* within line tab rendering
* xyz text rendering
* svg text rendering, markers, lab plot text rotation

