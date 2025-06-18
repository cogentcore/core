+++
Categories = ["Concepts"]
+++

A **sprite** is a is a top-level rendering element similar to a [[canvas]] that paints onto a transparent layer over the entire window that is cleared every [[render]] pass. Sprites are used for text cursors in [[text field]] and [[text editor]], and for [[drag and drop]]. To support cursor sprites and other animations, the sprites are redrawn at a minimum update rate that is at least as fast as [[doc:core.SystemSettingsData.CursorBlinkTime]].

Sprites can also receive mouse [[event]]s, within their event bounding box, so they can be used for dynamic control functions, like the "handles" in a drawing program, as in the [Cogent Canvas](https://cogentcore.org/cogent/canvas) app.

The primary features of sprites are:

* They are *not* under control of the [[layout]] system, and can be placed anywhere. That is because they are *not* [[widget]]s, they are simpler rendering elements. See [[doc:core.Sprite]].

* The sprite layer is cleared every time and overlaid on top of any other content, whereas the [[scene]] normally accumulates drawing over time so that different widget-regions can be focally updated without having to redraw the entire scene every time.
