# GiSt

GiSt ("gist") contains all the styling information for the GoGi system.

These are all based on the CSS standard: https://www.w3schools.com/cssref/default.asp

The `xml` struct tags provide the (lowercase) keyword for each tag -- tags can be parsed from strings using these keywords, and set by GoKi `ki.Prop` values, based on code in `style_props.go` and `paint_props.go`.  That props code must be updated if style fields are added.

This was factored out from the `gi` package for version 1.1.0, making it easier to navigate the code and keeping `gi` smaller and consisting mostly of `gi.Node2D` widgets.

Minor updates are required to rename `gi.` -> `gist.` for style properties that are set literally and not using string names (which translate directly), and any references to `gi.Color` -> `gist.Color`

# TODO

There are several remaining styles that were not yet implemented or defined yet, many dealing with international text, and other forms of layout or rendering that are not yet implemented.

