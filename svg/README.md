# svg

SVG implements the Scalable Vector Graphics standard, rendering onto a Go standard `image` bitmap.

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
