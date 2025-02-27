# svg

SVG implements the Scalable Vector Graphics standard, using core `tree` nodes to provide a structured in-memory representation of the SVG elements, suitable for subsequent manipulation and editing, as in the Cogent Canvas SVG drawing program.

## svg cli

Cogent Core provides a helpful command line tool for rendering and creating svg files. To install it, run:

```sh
go install cogentcore.org/core/svg/cmd/svg@main
```

Example usage:

```sh
# Render an svg file
svg icon.svg 250
# Embed an image into an svg file
svg embed-image image.png
```

## Technical Details

The SVG "scene" is processed in 3 passes:

* `Style` applies CSS-based styling properties to the `styles.Paint` style object on each node, using inheritance etc. These properties are set originally from xml parsing, and due to the nature of that process, it is not possible to use the styler functions used for core widgets.

* `BBoxes` computes bounding boxes for each element, based on local-unit `LocalBBox` values that are then transformed into final rendering pixel units. `Group` elements compute the union over their children. `Text` does layout of the `shaped.Lines` text at this point.

* `Render` actually makes the `paint.Painter` rendering calls, only for visible elements.

