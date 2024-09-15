# table 

[![Go Reference](https://pkg.go.dev/badge/cogentcore.org/core/table.svg)](https://pkg.go.dev/cogentcore.org/core/table)

**table** provides a DataTable / DataFrame structure similar to [pandas](https://pandas.pydata.org/) and [xarray](http://xarray.pydata.org/en/stable/) in Python, and [Apache Arrow Table](https://github.com/apache/arrow/tree/master/go/arrow/array/table.go), using [tensor](../tensor) n-dimensional columns aligned by common outermost row dimension.

See [examples/dataproc](examples/dataproc) for a demo of how to use this system for data analysis, paralleling the example in [Python Data Science](https://jakevdp.github.io/PythonDataScienceHandbook/03.08-aggregation-and-grouping.html) using pandas, to see directly how that translates into this framework.

Whereas an individual `Tensor` can only hold one data type, the `Table` allows coordinated storage and processing of heterogeneous data types, aligned by the outermost row dimension. The main `tensor` data processing functions are defined on the individual tensors (which are the universal computational element in the `tensor` system), but the coordinated row-wise indexing in the table is important for sorting or filtering a collection of data in the same way, and grouping data by a common set of "splits" for data analysis.  Plotting is also driven by the table, with one column providing a shared X axis for the rest of the columns.

The `Table` mainly provides "infrastructure" methods for adding tensor columns and CSV (comma separated values, and related tab separated values, TSV) file reading and writing.  Any function that can be performed on an individual column should be done using the `tensor.Indexed` and `Tensor` methods directly.

As a general convention, it is safest, clearest, and quite fast to access columns by name instead of index (there is a `map` from name to index), so the base access method names generally take a column name argument, and those that take a column index have an `Index` suffix.

The table itself stores raw data `tensor.Tensor` values, and the `Column` (by name) and `ColumnIndex` methods return a `tensor.Indexed` with the `Indexes` pointing to the shared table-wide `Indexes` (which can be `nil` if standard sequential order is being used).  

If you call Sort, Filter or other routines on an individual column tensor, then you can grab the updated indexes via the `IndexesFromTensor` method so that they apply to the entire table.

There are also multi-column `Sort` and `Filter` methods on the Table itself.

It is very low-cost to create a new View of an existing Table, via `NewTableView`, as they can share the underlying `Columns` data.

# Cheat Sheet

`dt` is the Table pointer variable for examples below:

## Table Access

Column data access:

```Go
// FloatRowCell is method on the `tensor.Indexed` returned from `Column` method.
// This is the best method to use in general for generic 1D data access,
// as it works on any data from 1D on up (although it only samples the first value
// from higher dimensional data) .
val := dt.Column("Values").FloatRowCell(3, 0)
```

```Go
dt.Column("Name").SetStringRowCell(4, 0)
```

todo: more

## Sorting and Filtering

## Splits ("pivot tables" etc), Aggregation

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




