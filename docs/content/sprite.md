+++
Categories = ["Concepts"]
+++

A **sprite** is a is a top-level rendering element similar to a [[canvas]] that paints onto a transparent layer over the entire window that is cleared every render pass. Sprites are used for text cursors in the [[text field]] and [[text editor]] and for [[drag and drop]]. To support cursor sprites and other animations, the sprites are redrawn at a minimum update rate that is at least as fast as CursorBlinkTime.

Sprites can also receive mouse [[events]], within their event bounding box, so they can be used for dynamic control functions, like the "handles" in a drawing program, as in the [Cogent Canvas](https://github.com/cogentcore/cogent/canvas) app.

The primary features of sprites are:

* They are not under control of the [[layout]] system, and can be placed anywhere.

* The sprite layer is cleared every time and overlaid on top of any other content, whereas the [[scene]] normally accumulates drawing over time so that different widget-regions can be focally updated without having to redraw the entire scene every time.

