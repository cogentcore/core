// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clust

import (
	"fmt"
	"testing"

	"cogentcore.org/core/tensor/stats/metric"
	"cogentcore.org/core/tensor/stats/simat"
	"cogentcore.org/core/tensor/table"
)

func TestClust(t *testing.T) {
	dt := &table.Table{}
	err := dt.OpenCSV("test_data/faces.dat", table.Tab)
	if err != nil {
		t.Error(err)
	}
	ix := table.NewIndexView(dt)
	smat := &simat.SimMat{}
	smat.TableCol(ix, "Input", "Name", false, metric.Euclidean64)

	// fmt.Printf("%v\n", smat.Mat)
	// cl := Glom(smat, MinDist)
	cl := Glom(smat, AvgDist)
	s := cl.Sprint(smat, 0)
	fmt.Println(s)
}
