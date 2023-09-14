# GiRl 

GiRl is the GoGi render library -- renders to an `image.RGBA` using styles defined in GiSt styles.

The original rendering design borrowed heavily from: https://github.com/fogleman/gg.  Relative to gg, the primary advantages of GiRl include:

* Use of https://github.com/srwiley/rasterx and SVG / CSS styles to provide more SVG-compliant rendering, that is also fast.  Supports color gradients etc.

* Full-featured HTML-styled text layout and rendering, which is significantly more advanced than what is present in the gg system.

See [svg](https://github.com/goki/svg) package for an SVG rendering framework built on top of GiRl.

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


# GiSt

GiSt ("gist") contains all the styling information for the GoGi system.

These are all based on the CSS standard: https://www.w3schools.com/cssref/default.asp

The `xml` struct tags provide the (lowercase) keyword for each tag -- tags can be parsed from strings using these keywords, and set by GoKi `ki.Prop` values, based on code in `style_props.go` and `paint_props.go`.  That props code must be updated if style fields are added.

This was factored out from the `gi` package for version 1.1.0, making it easier to navigate the code and keeping `gi` smaller and consisting mostly of `gi.Node2D` widgets.

Minor updates are required to rename `gi.` -> `gist.` for style properties that are set literally and not using string names (which translate directly), and any references to `gi.Color` -> `gist.Color`

# TODO

There are several remaining styles that were not yet implemented or defined yet, many dealing with international text, and other forms of layout or rendering that are not yet implemented.

