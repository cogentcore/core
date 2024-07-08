# colors

Package colors provides named colors, utilities for manipulating colors, and Material Design 3 color schemes, palettes, and keys in Go.

## image.Image as a universal color

The Go standard library defines the `image.Image` interface, which returns a color at a given x,y coordinate via the `At(x,y) color.Color` method.  This provides the most general way of specifying a color, encompassing everything from a single solid color to a pattern to a gradient to an actual image.  Thus, `image.Image` is used to specify colors in most places in the Cogent Core system.

* `image.Uniform` always returns a single uniform color, ignoring the coordinates. Use the `colors.Uniform` helper function to create a new uniform color (it just returns `image.NewUniform(c)`).

* `gradient.Gradient` (from `colors/gradient`) is an `image.Image` interface that specifies an SVG-compatible color gradient using Stops to define specific points of color, with the specific color at each point as a proportional blend between the two nearest stops.  There are `gradient.Linear` and `gradient.Radial` subtypes.

## gradient.Applier

We often need to apply opacity transformations to colors, which have the effect of darkening or lightening colors, for example indicating different states, such as when a Button is hovered vs. clicked.  To do this efficiently and flexibly for the different types of `image.Image` colors, the `gradient` package defines an `ApplyFunc` function that takes a color and returns a modified color.

There is an `gradient.Applier` that implements the `image.Image` interface (so it can be used for any such color), which applies the color transformation via the `At(x,y)` method, so it automatically transforms the color output of any source image (where the `image.Image` is an embedded field) of the struct type.

Finally, there is an `ApplyOpacity` method that is extra efficient for the Uniform and Gradient cases, directly transforming the uniform color or color of the Stops in the gradient, to avoid the extra ApplyFunc call (which is used for the general case of an actual image).

The reason this Apply logic is in `gradient` is so it can manage the Gradient case.

