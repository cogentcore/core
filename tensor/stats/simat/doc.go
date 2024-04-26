// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package simat provides similarity / distance matrix functions that create
a SimMat matrix from Tensor or Table data.  Any metric function defined
in metric package (or user-created) can be used.

The SimMat contains the Tensor of the similarity matrix values, and
labels for the Rows and Columns.

The etview package provides a SimMatGrid widget that displays the SimMat
with the labels.
*/
package simat
