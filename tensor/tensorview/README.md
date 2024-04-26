# tensorview

Docs: [GoDoc](https://pkg.go.dev/cogentcore.org/core/tensor/tensorview)

`tensorview` provides GUI Views of `table.Table` and `tensor.Tensor` structures using the [Core](https://cogentcore.org/core/views) View framework, as Cogent Core Widgets.

Add this to `import` to get these views to be registered with the Cogent Core Value system:

```Go
	_ "cogentcore.org/core/tensor/tensorview" // include to get gui views
```

* `TableView` provides a row-and-column tabular GUI interface, similar to a spreadsheet, for viewing and editing Table data.  Any higher-dimensional tensor columns are shown as TensorGrid elements that can be clicked to open a TensorView editor with actual numeric values in a similar spreadsheet-like GUI.

* `TensorView` provides a spreadsheet-like GUI for viewing and editing tensor data.

* `TensorGrid` provides a 2D colored grid display of tensor data, collapsing any higher dimensions down to 2D.  Different views.ColorMaps can be used to translate values into colors.

