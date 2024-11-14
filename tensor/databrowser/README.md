# databrowser

The databrowser package provides GUI elements for data exploration and visualization, and a simple `Browser` implementation that combines these elements.

* `FileTree` (with `FileNode` elements), implementing a [filetree](https://github.com/cogentcore/tree/main/filetree) that has support for a [datafs](../datafs) filesystem, and data files in an actual filesystem. It has a `Tabber` pointer that handles the viewing actions on `datafs` elements (showing a Plot, etc).

* `Tabber` interface and `Tabs` base implementation provides methods for showing data plots and editors in tabs.

* `Terminal` running a `goal` shell that supports interactive commands operating on the `datafs` data etc. TODO!

The basic `Browser` puts the `FileTree` in a left `Splits` and the `Tabs` in the right, and supports interactive exploration and visualization of data.

In the [emergent](https://github.com/emer) framework, these elements are combined with other GUI elements to provide a full neural network simulation environment on top of the databrowser foundation.

