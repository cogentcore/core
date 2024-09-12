# table 

[![Go Reference](https://pkg.go.dev/badge/cogentcore.org/core/table.svg)](https://pkg.go.dev/cogentcore.org/core/table)

**table** provides a DataTable / DataFrame structure similar to [pandas](https://pandas.pydata.org/) and [xarray](http://xarray.pydata.org/en/stable/) in Python, and [Apache Arrow Table](https://github.com/apache/arrow/tree/master/go/arrow/array/table.go), using [tensor](../tensor) n-dimensional columns aligned by common outermost row dimension.

See [examples/dataproc](examples/dataproc) for a demo of how to use this system for data analysis, paralleling the example in [Python Data Science](https://jakevdp.github.io/PythonDataScienceHandbook/03.08-aggregation-and-grouping.html) using pandas, to see directly how that translates into this framework.

Whereas an individual `Tensor` can only hold one data type, the `table` allows coordinated storage and processing of heterogeneous data types, aligned by the outermost row dimension. While the main data processing functions are defined on the individual tensors (which are the universal computational element in the `tensor` system), the coordinated row-wise indexing in the table is important for sorting or filtering a collection of data in the same way, and grouping data by a common set of "splits" for data analysis.  Plotting is also driven by the table, with one column providing a shared X axis for the rest of the columns.

As a general convention, it is safest, clearest, and quite fast to access columns by name instead of index (there is a map that caches the column indexes), so the base access method names generally take a column name argument, and those that take a column index have an `Index` suffix.

The table itself stores raw data `tensor.Tensor` values, and the `Column` (by index) and `ColumnByName` methods return a `tensor.Indexed` with the `Indexes` pointing to the shared table-wide `Indexes` (which can be `nil` if standard sequential order is being used).  It is best to use the table-wise `Sort` and `Filter` methods (and any others that affect the indexes) to ensure the indexes are properly coordinated.  Resetting the column tensor indexes to `nil` (via the `Sequential` method) will break any connection to the table indexes, so that any subsequent index-altering operations on that indexed tensor will be fine.

# Cheat Sheet

`dt` is the etable pointer variable for examples below:

## Table Access

Scalar columns:

```Go
val := dt.Float("ColName", row)
```

```Go
str := dt.StringValue("ColName", row)
```

Tensor (higher-dimensional) columns:

```Go
tsr := dt.Tensor("ColName", row) // entire tensor at cell (a row-level SubSpace of column tensor)
```

```Go
val := dt.TensorFloat1D("ColName", row, cellidx) // idx is 1D index into cell tensor
```

## Set Table Value

```Go
dt.SetFloat("ColName", row, val)
```

```Go
dt.SetString("ColName", row, str)
```

Tensor (higher-dimensional) columns:

```Go
dt.SetTensor("ColName", row, tsr) // set entire tensor at cell 
```

```Go
dt.SetTensorFloat1D("ColName", row, cellidx, val) // idx is 1D index into cell tensor
```

## Find Value(s) in Column

Returns all rows where value matches given value, in string form (any number will convert to a string)

```Go
rows := dt.RowsByString("ColName", "value", etable.Contains, etable.IgnoreCase)
```

Other options are `etable.Equals` instead of `Contains` to search for an exact full string, and `etable.UseCase` if case should be used instead of ignored.

## Index Views (Sort, Filter, etc)

The [Indexed](https://godoc.org/github.com/goki/etable/v2/etable#Indexed) provides a list of row-wise indexes into a table, and Sorting, Filtering and Splitting all operate on this index view without changing the underlying table data, for maximum efficiency and flexibility.

```Go
ix := etable.NewIndexed(et) // new view with all rows
```

### Sort

```Go
ix.SortColumnName("Name", etable.Ascending) // etable.Ascending or etable.Descending
SortedTable := ix.NewTable() // turn an Indexed back into a new Table organized in order of indexes
```

or:

```Go
nmcl := dt.ColumnByName("Name") // nmcl is an etensor of the Name column, cached
ix.Sort(func(t *Table, i, j int) bool {
	return nmcl.StringValue1D(i) < nmcl.StringValue1D(j)
})
```

### Filter

```Go
nmcl := dt.ColumnByName("Name") // column we're filtering on
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
ix := etable.NewIndexed(et) // new view with all rows
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




