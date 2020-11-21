# GiRl 

GiRl is the GoGi render library -- renders to an `image.RGBA` using styles defined in GiSt styles.

The original rendering design borrowed heavily from: https://github.com/fogleman/gg

And was subsequently integrated with https://github.com/srwiley/rasterx which, provides SVG compliant and fast rendering.

The `svg` package provides Node types, which can be created by parsing a standard SVG file, that use this rendering library to render standard SVG images.  These are used for the icons in GoGi.

The basic structure for rendering is in two objects:

* `girl.Paint` has the styles, and methods for drawing.

* `girl.State` has all the underlying rendering state, and the `Paint` methods take state as the first arg.

This is useful for allowing multiple different painters to all render onto a common back-end image controlled by the `girl.State` object.  In GoGi, the `Viewport2D` has the shared `girl.State`, while each Widget or SVG Node has its own `girl.Paint` object and styles.

It would be easy to use GiRl separate from the broader GoGi system -- there are just two "hooks" that need to be set, both in the `gist` style system: 

* `ThePrefs` needs to be set a preferences object that satisfies the `Prefs` interface to provide user-settable preferences (just for colors and fonts).

* `Context` is passed to styling functions that for some color names that refer to a current color -- it is optional and will be ignored if nil.

In addition, the styles need a properly initialized `units.Context` to be passed to the `ToDots` method that converts all the various `units` into concrete physical pixels that will be rendered.

# Text layout

The complex task of laying out text is handled by the `girl.Text` system, which has spans of runes, with full styling information that handles all the standard HTML-style markup, including underlining, super / subscripts, rotation, etc.

This framework is sufficient for all forms of bidirectional / vertical text layout etc, but only the left-right case has been implemented at this point.

Styling, Formatting / Layout, and Rendering are each handled separately as three different levels in the stack -- this simplifies many things to separate in this way, and makes the final render pass maximally efficient and high-performance, at the potential cost of some memory redundancy.

