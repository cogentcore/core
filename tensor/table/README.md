# table 

[![Go Reference](https://pkg.go.dev/badge/cogentcore.org/core/table.svg)](https://pkg.go.dev/cogentcore.org/core/table)

**table** provides a DataTable / DataFrame structure similar to [pandas](https://pandas.pydata.org/) and [xarray](http://xarray.pydata.org/en/stable/) in Python, and [Apache Arrow Table](https://github.com/apache/arrow/tree/master/go/arrow/array/table.go), using [tensor](../tensor) n-dimensional columns aligned by common outermost row dimension.

See [examples/dataproc](examples/dataproc) for a demo of how to use this system for data analysis, paralleling the example in [Python Data Science](https://jakevdp.github.io/PythonDataScienceHandbook/03.08-aggregation-and-grouping.html) using pandas, to see directly how that translates into this framework.

As a general convention, it is safest, clearest, and quite fast to access columns by name instead of index (there is a map that caches the column indexes), so the base access method names generally take a column name argument, and those that take a column index have an `Index` suffix.  In addition, we use the `Try` suffix for versions that return an error message.  It is a bit painful for the writer of these methods but very convenient for the users.

The following packages are included:

* [bitslice](bitslice) is a Go slice of bytes `[]byte` that has methods for setting individual bits, as if it was a slice of bools, while being 8x more memory efficient.  This is used for encoding null entries in  `etensor`, and as a Tensor of bool / bits there as well, and is generally very useful for binary (boolean) data.

* [etensor](etensor) is a Tensor (n-dimensional array) object.  `etensor.Tensor` is an interface that applies to many different type-specific instances, such as `etensor.Float32`.  A tensor is just a `etensor.Shape` plus a slice holding the specific data type.  Our tensor is based directly on the [Apache Arrow](https://github.com/apache/arrow/tree/master/go) project's tensor, and it fully interoperates with it.  Arrow tensors are designed to be read-only, and we needed some extra support to make our `etable.Table` work well, so we had to roll our own.  Our tensors also interoperate fully with Gonum's 2D-specific Matrix type for the 2D case.

* [etable](etable) has the `etable.Table` DataTable / DataFrame object, which is useful for many different data analysis and database functions, and also for holding patterns to present to a neural network, and logs of output from the models, etc.  A `etable.Table` is just a slice of `etensor.Tensor` columns, that are all aligned along the outer-most *row* dimension.  Index-based indirection, which is essential for efficient Sort, Filter etc, is provided by the `etable.IndexView` type, which is an indexed view into a Table.  All data processing operations are defined on the IndexView.

* [eplot](eplot) provides an interactive 2D plotting GUI in [GoGi](https://cogentcore.org/core/gi) for Table data, using the [gonum plot](https://github.com/gonum/plot) plotting package.  You can select which columns to plot and specify various basic plot parameters.

* [tensorcore](tensorcore) provides an interactive tabular, spreadsheet-style GUI using [GoGi](https://cogentcore.org/core/gi) for viewing and editing `etable.Table` and `etable.Tensor` objects.  The `tensorcore.TensorGrid` also provides a colored grid display higher-dimensional tensor data.

* [agg](agg) provides standard aggregation functions (`Sum`, `Mean`, `Var`, `Std` etc) operating over `etable.IndexView` views of Table data.  It also defines standard `AggFunc` functions such as `SumFunc` which can be used for `Agg` functions on either a Tensor or IndexView.

* [tsragg](tsragg) provides the same agg functions as in `agg`, but operating on all the values in a given `Tensor`.  Because of the indexed, row-based nature of tensors in a Table, these are not the same as the `agg` functions.

* [split](split) supports splitting a Table into any number of indexed sub-views and aggregating over those (i.e., pivot tables), grouping, summarizing data, etc.

* [metric](metric) provides similarity / distance metrics such as `Euclidean`, `Cosine`, or `Correlation` that operate on slices of `[]float64` or `[]float32`.

* [simat](simat) provides similarity / distance matrix computation methods operating on `etensor.Tensor` or `etable.Table` data.  The `SimMat` type holds the resulting matrix and labels for the rows and columns, which has a special `SimMatGrid` view in `etview` for visualizing labeled similarity matricies.

* [pca](pca) provides principal-components-analysis (PCA) and covariance matrix computation functions.

* [clust](clust) provides standard agglomerative hierarchical clustering including ability to plot results in an eplot.

* [minmax](minmax) is home of basic Min / Max range struct, and `norm` has lots of good functions for computing standard norms and normalizing vectors.

* [utils](utils) has various table-related utility command-line utility tools, including `etcat` which combines multiple table files into one file, including option for averaging column data.

# Cheat Sheet

`et` is the etable pointer variable for examples below:

## Table Access

Scalar columns:

```Go
val := et.Float("ColName", row)
```

```Go
str := et.StringValue("ColName", row)
```

Tensor (higher-dimensional) columns:

```Go
tsr := et.Tensor("ColName", row) // entire tensor at cell (a row-level SubSpace of column tensor)
```

```Go
val := et.TensorFloat1D("ColName", row, cellidx) // idx is 1D index into cell tensor
```

## Set Table Value

```Go
et.SetFloat("ColName", row, val)
```

```Go
et.SetString("ColName", row, str)
```

Tensor (higher-dimensional) columns:

```Go
et.SetTensor("ColName", row, tsr) // set entire tensor at cell 
```

```Go
et.SetTensorFloat1D("ColName", row, cellidx, val) // idx is 1D index into cell tensor
```

## Find Value(s) in Column

Returns all rows where value matches given value, in string form (any number will convert to a string)

```Go
rows := et.RowsByString("ColName", "value", etable.Contains, etable.IgnoreCase)
```

Other options are `etable.Equals` instead of `Contains` to search for an exact full string, and `etable.UseCase` if case should be used instead of ignored.

## Index Views (Sort, Filter, etc)

The [IndexView](https://godoc.org/github.com/goki/etable/v2/etable#IndexView) provides a list of row-wise indexes into a table, and Sorting, Filtering and Splitting all operate on this index view without changing the underlying table data, for maximum efficiency and flexibility.

```Go
ix := etable.NewIndexView(et) // new view with all rows
```

### Sort

```Go
ix.SortColumnName("Name", etable.Ascending) // etable.Ascending or etable.Descending
SortedTable := ix.NewTable() // turn an IndexView back into a new Table organized in order of indexes
```

or:

```Go
nmcl := et.ColumnByName("Name") // nmcl is an etensor of the Name column, cached
ix.Sort(func(t *Table, i, j int) bool {
	return nmcl.StringValue1D(i) < nmcl.StringValue1D(j)
})
```

### Filter

```Go
nmcl := et.ColumnByName("Name") // column we're filtering on
ix.Filter(func(t *Table, row int) bool {
	// filter return value is for what to *keep* (=true), not exclude
	// here we keep any row with a name that contains the string "in"
	return strings.Contains(nmcl.StringValue1D(row), "in")
})
```

### Splits ("pivot tables" etc), Aggregation

Create a table of mean values of "Data" column grouped by unique entries in "Name" column, resulting table will be called "DataMean":

```Go
byNm := split.GroupBy(ix, []string{"Name"}) // column name(s) to group by
split.Agg(byNm, "Data", agg.AggMean) // 
gps := byNm.AggsToTable(etable.AddAggName) // etable.AddAggName or etable.ColNameOnly for naming cols
```

Describe (basic stats) all columns in a table:

```Go
ix := etable.NewIndexView(et) // new view with all rows
desc := agg.DescAll(ix) // summary stats of all columns
// get value at given column name (from original table), row "Mean"
mean := desc.Float("ColNm", desc.RowsByString("Agg", "Mean", etable.Equals, etable.UseCase)[0])
```

# CSV / TSV file format

Tables can be saved and loaded from CSV (comma separated values) or TSV (tab separated values) files.  See the next section for special formatting of header strings in these files to record the type and tensor cell shapes.

## Type and Tensor Headers

To capture the type and shape of the columns, we support the following header formatting.  We weren't able to find any other widely supported standard (please let us know if there is one that we've missed!)

Here is the mapping of special header prefix characters to standard types:
```Go
'$': etensor.STRING,
'%': etensor.FLOAT32,
'#': etensor.FLOAT64,
'|': etensor.INT64,
'@': etensor.UINT8,
'^': etensor.BOOl,
```

Columns that have tensor cell shapes (not just scalars) are marked as such with the *first* such column having a `<ndim:dim,dim..>` suffix indicating the shape of the *cells* in this column, e.g., `<2:5,4>` indicates a 2D cell Y=5,X=4.  Each individual column is then indexed as `[ndims:x,y..]` e.g., the first would be `[2:0,0]`, then `[2:0,1]` etc.

### Example

Here's a TSV file for a scalar String column (`Name`), a 2D 1x4 tensor float32 column (`Input`), and a 2D 1x2 float32 `Output` column.

```
_H:	$Name	%Input[2:0,0]<2:1,4>	%Input[2:0,1]	%Input[2:0,2]	%Input[2:0,3]	%Output[2:0,0]<2:1,2>	%Output[2:0,1]
_D:	Event_0	1	0	0	0	1	0
_D:	Event_1	0	1	0	0	1	0
_D:	Event_2	0	0	1	0	0	1
_D:	Event_3	0	0	0	1	0	1
```




