# textcore Editor

The `textcore.Base` provides a base implementation for a core widget that views `lines.Lines` text content.

A critical design feature is that the Base widget can switch efficiently among different `lines.Lines` content. For example, in Cogent Code there are 2 editor widgets that are reused for all files, including viewing the same file across both editors. Thus, all of the state comes from the underlying Lines buffers.

The Lines handles all layout and markup styling, so the Base just needs to render the results of that. Thus, there is no need for the Editor to ever drive a NeedsLayout call itself: everything is handled in the Render step, including the presence or absence of the scrollbar, which is a little bit complicated because adding a scrollbar changes the effective width and thus the internal layout.

# TODO

* dynamic scroll re-layout
* link handling
* outputbuffer formatting
* within text tabs
* xyz text rendering
* lab/plot rendering

