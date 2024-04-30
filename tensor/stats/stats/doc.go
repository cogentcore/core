// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package agg provides aggregation functions operating on IndexView indexed views
of table.Table data, along with standard AggFunc functions that can be used
at any level of aggregation from tensor on up.

The main functions use names to specify columns, and *Index and *Try versions
are available that operate on column indexes and return errors, respectively.

See tsragg package for functions that operate directly on a tensor.Tensor
without the indexview indirection.
*/
package stats
