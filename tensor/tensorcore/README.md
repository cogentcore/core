# tensorcore

`tensorcore` provides GUI Views of `table.Table` and `tensor.Tensor` structures as Cogent Core Widgets.

Add this to `import` to get these views to be registered with the Cogent Core Value system:

```Go
	_ "cogentcore.org/core/tensor/tensorcore" // include to get GUI views
```

* `Table` provides a row-and-column tabular GUI interface, similar to a spreadsheet, for viewing and editing Table data.  Any higher-dimensional tensor columns are shown as TensorGrid elements that can be clicked to open a TensorView editor with actual numeric values in a similar spreadsheet-like GUI.

* `TensorView` provides a spreadsheet-like GUI for viewing and editing tensor data.

* `TensorGrid` provides a 2D colored grid display of tensor data, collapsing any higher dimensions down to 2D.  Different core.ColorMaps can be used to translate values into colors.

