# GiRl 

GiRl is the GoGi render library -- renders to an `image.RGBA` using CSS-based styling parameters.

The original rendering design borrowed heavily from: [gg](https://github.com/fogleman/gg).  Relative to gg, the primary advantages of GiRl include:

* Use of [rasterx](https://github.com/srwiley/rasterx) and SVG / CSS styles to provide more SVG-compliant rendering, that is also fast.  Supports color gradients etc.

* Full-featured HTML-styled text layout and rendering, which is significantly more advanced than what is present in the gg system.

See [svg](https://github.com/goki/svg) package for an SVG rendering framework built on top of GiRl.

There packages within GiRl are:

* [styles](styles): has all the CSS styling parameters for general HTML-level styling (`styles.Style` type) and for SVG rendering (`styles.Paint`)

* [units](units): supports all the standard CSS unit types (`em`, `px`, `pt` etc) and conversion into an actual rendering-level `dot` scale that reflects the actual physical pixel dots (`px` has been co-opted into a 96 DPI unit).

* [states](states): supports CSS-standard states for GUI elements, based on [CSS Pseudo-classes](https://developer.mozilla.org/en-US/docs/Web/CSS/Pseudo-classes)

* [paint](paint): has all the actual rendering methods (lines, curves, fill etc) added to its version of the `Paint` type, which embeds `styles.Paint`.

Painting involves two separate classes, enabling the separation between style and state (missing in the `gg` package):

* `paint.Paint` has the styles, and methods for drawing.

* `paint.State` has all the underlying rendering state, and the `Paint` methods take `State` as the first arg.

This is useful for allowing multiple different painters to all render onto a common back-end image controlled by the `paint.State` object.  In the GoGi GUI, the `gi.Scene` has the shared `paint.State`, while each Widget or SVG Node has its own `paint.Paint` object and styles.

There are two global variables that need to be set, both in the `styles` system: 

* `ThePrefs` needs to be set a preferences object that satisfies the `Prefs` interface to provide user-settable preferences (just for colors and fonts).

* `Context` is passed to styling functions that for some color names that refer to a current color -- it is optional and will be ignored if nil.

In addition, the styles need a properly initialized `units.Context` to be passed to the `ToDots` method that converts all the various `units` into concrete physical pixels that will be rendered.

# Text layout

The complex task of laying out text is handled by the `girl.Text` system, which has spans of runes, with full styling information that handles all the standard HTML-style markup, including underlining, super / subscripts, rotation, etc.

This framework is sufficient for all forms of bidirectional / vertical text layout etc, but only the left-right case has been implemented at this point.

Styling, Formatting / Layout, and Rendering are each handled separately as three different levels in the stack -- this simplifies many things to separate in this way, and makes the final render pass maximally efficient and high-performance, at the potential cost of some memory redundancy.


# Styles

Styles are based on the CSS standard: https://www.w3schools.com/cssref/default.asp

The `xml` struct tags show the (lowercase) keyword for each tag -- tags can be parsed from strings using these keywords, and set by `map[string]any` "props" property values, based on code in `style_props.go` and `paint_props.go`.

# TODO

There are several remaining styles that were not yet implemented or defined yet, many dealing with international text, and other forms of layout or rendering that are not yet implemented.

