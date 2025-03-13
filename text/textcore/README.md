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

* Spinner plus buttons sometimes erroneously render black backgrounds, especially when scrolling in table in demo

* tweak underlining on web? looks strange (not far down enough)

* scroll animation uses new Animator struct objs that have a func that is called in Paint loop, use delta-dist / dt^2 accelleration factor for remaining scroll drift

* even fixed elements with no scroll can be pulled down with scrolling, for the "pull down refresh" action; animation restores to prior location

* dialog shadows are accumulating: BlitBox nil image is not filling most likely.

* we need to fix line width on our core.Canvas and cleanup junk in htmlcanvas code.

* web: do xyz and video sources.

* Text editor is not rendering a lot of stuff on Retina display: base.onsurface color is not updating -- need to figure out why.

* web Border width too small on many widgets on Retina: need DPR multiplier.

* High precision text rendering on web?  Kai do benchmarks.

* move shaper to renderwindow so popup menus etc don't need to make their own? SVG too!? is every icon getting a shaper?

* svg marker glitch is last remaining bug: debugit!

* SVG, PDF backends

* textcore base test horizontal scrolling

* prompt on quitting modified file hangs: can't figure it out: dialog is called but never opens, then it hangs

* shaped does not process `\n` https://github.com/go-text/typesetting/issues/185 

### Lowpri

* better job finding path fragments from file links -- iteratively try stuff.

* check for negative advance and highlighting issues / tests

* emoji, svg, bitmap font rendering: could not get color emoji to work

* code newFiles AddToVCS should default on -- not working

* TestMarkupSpellErr: still some rich tag format issues but mostly working.. why is this so hard!?

* cleanup unused base stuff

* text render highlight region fill in blanks better: hard b/c at run level, doesn't have context.

* xyz physics GrabEyeImg causes crashing with goroutine renderAsync in renderwindow, but otherwise is ok

* code: Markup colors are baked in when output is generated -- no remarkup possible!


