# Plot

The `plot` package generates 2D plots of data using the Cogent Core `paint` rendering system.  The `plotcore` sub-package has Cogent Core Widgets that can be used in applications.  
* `Plot` is just a wrapper around a `plot.Plot`, for manually-configured plots.
* `PlotEditor` is an interactive plot viewer that supports selection of which data to plot, and configuration of many plot parameters.

The code is adapted from the [gonum plot](https://github.com/gonum/plot) package (which in turn was adapted from google's [plotinum](https://code.google.com/archive/p/plotinum/), to use the Cogent Core [styles](../styles) and [paint](../paint) rendering framework, which also supports SVG output of the rendering.

Here is the copyright notice for that package:
```go
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
```

