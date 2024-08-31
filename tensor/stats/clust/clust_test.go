// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clust

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"cogentcore.org/core/tensor/stats/metric"
	"cogentcore.org/core/tensor/stats/simat"
	"cogentcore.org/core/tensor/table"
)

var clustres = `
0: 
	9.181170003996987: 
		5.534356399283666: 
			4.859933131085473: 
				3.4641016151377544: Mark_sad Mark_happy 
				3.4641016151377544: Zane_sad Zane_happy 
			3.4641016151377544: Alberto_sad Alberto_happy 
		5.111664626761644: 
			4.640135790634417: 
				4: Lisa_sad Lisa_happy 
				3.4641016151377544: Betty_sad Betty_happy 
			3.605551275463989: Wendy_sad Wendy_happy `

func TestClust(t *testing.T) {
	dt := &table.Table{}
	err := dt.OpenCSV("testdata/faces.dat", table.Tab)
	if err != nil {
		t.Error(err)
	}
	ix := table.NewIndexView(dt)
	smat := &simat.SimMat{}
	smat.TableColumn(ix, "Input", "Name", false, metric.Euclidean64)

	// fmt.Printf("%v\n", smat.Mat)
	// cl := Glom(smat, MinDist)
	cl := Glom(smat, AvgDist)
	// s := cl.Sprint(smat, 0)
	// fmt.Println(s)

	var dists []float64

	var gather func(n *Node)
	gather = func(n *Node) {
		dists = append(dists, n.Dist)
		for _, kn := range n.Kids {
			gather(kn)
		}
	}
	gather(cl)

	exdists := []float64{0, 9.181170003996987, 5.534356399283667, 4.859933131085473, 3.4641016151377544, 0, 0, 3.4641016151377544, 0, 0, 3.4641016151377544, 0, 0, 5.111664626761644, 4.640135790634417, 4, 0, 0, 3.4641016151377544, 0, 0, 3.605551275463989, 0, 0}

	tolassert.EqualTolSlice(t, exdists, dists, 1.0e-8)
}
