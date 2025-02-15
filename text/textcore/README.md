# textcore Editor

The `textcore.Editor` provides a base implementation for a core widget that views `lines.Lines` text content.

A critical design feature is that the Editor widget can switch efficiently among different `lines.Lines` content. For example, in the Cogent Code editor, there are 2 editor widgets that are reused for all files, including viewing the same file across both editors. Thus, all of the state comes from the underlying Lines buffers.


