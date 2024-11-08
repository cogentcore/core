# Plot

The `plot` package generates 2D plots of data using the Cogent Core `paint` rendering system.  The `plotcore` sub-package has Cogent Core Widgets that can be used in applications.  
* `Plot` is just a wrapper around a `plot.Plot`, for manually-configured plots.
* `PlotEditor` is an interactive plot viewer that supports selection of which data to plot, and configuration of many plot parameters.

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

# History

The code is adapted from the [gonum plot](https://github.com/gonum/plot) package (which in turn was adapted from google's [plotinum](https://code.google.com/archive/p/plotinum/), to use the Cogent Core [styles](../styles) and [paint](../paint) rendering framework, which also supports SVG output of the rendering.

Here is the copyright notice for that package:
```go
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
```

