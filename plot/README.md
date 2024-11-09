# Plot

The `plot` package generates 2D plots of data using the Cogent Core `paint` rendering system.  The `plotcore` sub-package has Cogent Core Widgets that can be used in applications.  
* `Plot` is just a wrapper around a `plot.Plot`, for code-generated plots.
* `PlotEditor` is an interactive plot viewer that supports selection of which data to plot, and GUI configuration of plot parameters.

`plot` is designed to work in two potentially-conflicting ways:
* Code-based creation of a specific plot with specific data.
* GUI-based configuration of plots based on a `tensor.Table` of data columns (via `PlotEditor`).

The GUI constraint requires a more systematic, factorial organization of the space of possible plot data and how it is organized to create a plot, so that it can be configured with a relatively simple set of GUI settings. The overall logic is as follows:
* The overall plot has a single shared range of X, Y and optionally Z coordinate ranges, that defines where a data value in any plot type is plotted. These ranges are set based on the DataRanger interface.
* Plot content is driven by `Plotter` elements that each consume one or more sets of data, which is provided by a `Data` interface that maps onto a minimal subset of the `tensor.Tensor` interface, so a tensor directly satisfies the interface.
* Each `Plotter` element can generally handle multiple different data elements, that are index-aligned. For example, the basic `XY` plotter requires `X` and `Y` Valuers, and optionally `Size` or `Color` Valuers that apply to the Point elements, while `Bar` gets at least a `Y` but also optionally a `High` Valuer for an error bar.

The table-driven plotting case uses a `Group` name along with the `Roles` type (`X`, `Y` etc) and Plotter type names to organize different plots based on `Style` settings.  Columns with the same Group name all provide data to the same plotter using their different Roles, making it easy to configure various statistical plots of multiple series of grouped data.

Different plotter types (including custom ones) are registered along with their accepted input roles, to allow any type of plot to be generated.

# Styling

`plot.Style` contains the full set of styling parameters, which can be set using Styler functions that are attached to individual plot elements (e.g., lines, points etc) that drive the content of what is actually plotted (based on the `Plotter` interface).

Each such plot element defines a `Styler` method, e.g.,:

```Go
plt := plot.NewPlot()
ln := plots.AddLine.Styler(func(s *plot.Style) {
    s.Plot.Title = "My Plot" // overall Plot styles
    s.Line.Color = colors.Uniform(colors.Red) // line-specific styles
})
plt.Add(ln)
```

The `Plot` field (of type `PlotStyle`) contains all the properties that apply to the plot as a whole. Each element can set these values, and they are applied in the order the elements are added, so the last one gets final say. Typically you want to just set these plot-level styles on one element only and avoid any conflicts.

The rest of the style properties (e.g., `Line`, `Point`) apply to the element in question. There are also some default plot-level settings in `Plot` that apply to all elements, and the plot-level styles are updated first, so in this way it is possible to have plot-wide settings applied from one styler, that affect all plots (e.g., the line width, and whether lines and / or points are plotted or not).

# Plot Types

The following are the builtin standard plot types, in the `plots` package:

## 1D and 2D XY Data

### XY

`XY` is the workhorse standard Plotter, taking at least `X` and `Y` inputs, and plotting lines and / or points at each X, Y point. 

Optionally `Size` and / or `Color` inputs can be provided, which apply to the points. Thus, by using a `Point.Shape` of `Ring` or `Circle`, you can create a bubble plot by providing Size and Color data.

### Bar

`Bar` takes `Y` inputs, and draws bars of corresponding height.

An optional `High` input can be provided to also plot error bars above each bar.

To create a plot with multiple error bars, multiple Bar Plotters are created, with `Style.Width` parameters that have a shared `Stride = 1 / number of bars` and `Offset` that increments for each bar added.  The `plots.NewBars` function handles this directly.

### ErrorBar

`XErrorBar` and `YErrorBar` take `X`, `Y`, `Low`, and `High` inputs, and draws an `I` shaped error bar at the X, Y coordinate with the error "handles" around it.

### Labels

`Labels` takes `X`, `Y` and `Labels` string inputs and plots labels at the given coordinates.

### Box

`Box` takes `X`, `Y` (median line), `U`, `V` (box first and 3rd quartile values), and `Low`, `High` (Min, Max) inputs, and renders a box plot with error bars.

### XFill, YFill

`XFill` and `YFill` are used to draw filled regions between pairs of X or Y points, using the `X`, `Y`, and `BoxLow`, `BoxHigh` values to specify the center point (X, Y) and the region below / left and above / right to fill around that central point.

XFill along with an XY line can be used to draw the equivalent of the [matplotlib fill_between](https://matplotlib.org/stable/plot_types/basic/fill_between.html#sphx-glr-plot-types-basic-fill-between-py) plot.

YFill can be used to draw the equivalent of the [matplotlib violin plot](https://matplotlib.org/stable/plot_types/stats/violin.html#sphx-glr-plot-types-stats-violin-py).

### Pie

TODO

## 2D Grid-based

### ColorGrid

Input = Values and X, Y size

### Contour

??

### Vector

X,Y,U,V

Quiver?

## 3D 

TODO: use math32 3D projection math and you can just take each 3d point and reduce to 2D. For stuff you want to actually be able to use in SVG, it needs to ultimately be 2D, so it makes sense to support basic versions here, including XYZ (points, lines), Bar3D, wireframe.

Could also have a separate plot3d package based on `xyz` that is true 3D for interactive 3D plots of surfaces or things that don't make sense in this more limited 2D world.

# Statistical plots

The `statplot` package provides functions taking `tensor` data that produce statistical plots of the data, including Quartiles (Box with Median, Quartile, Min, Max), Histogram (Bar), Violin (YFill), Range (XFill), Cluster... 

TODO: add a Data scatter that plots points to overlay on top of Violin or Box.

## Histogram

## Quartiles

## Violin

## Range

## Cluster

# History

The code is adapted from the [gonum plot](https://github.com/gonum/plot) package (which in turn was adapted from google's [plotinum](https://code.google.com/archive/p/plotinum/), to use the Cogent Core [styles](../styles) and [paint](../paint) rendering framework, which also supports SVG output of the rendering.

Here is the copyright notice for that package:
```go
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
```

# TODO

* Grid?

