// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package split

import (
	"fmt"
	"math"

	"cogentcore.org/core/tensor/table"
	"gonum.org/v1/gonum/floats"
)

// Permuted generates permuted random splits of table rows, using given list of probabilities,
// which will be normalized to sum to 1 (error returned if sum = 0)
// names are optional names for each split (e.g., Train, Test) which will be
// used to label the Values of the resulting Splits.
func Permuted(ix *table.IndexView, probs []float64, names []string) (*table.Splits, error) {
	if ix == nil || ix.Len() == 0 {
		return nil, fmt.Errorf("split.Random table is nil / empty")
	}
	np := len(probs)
	if len(names) > 0 && len(names) != np {
		return nil, fmt.Errorf("split.Random names not same len as probs")
	}
	sum := floats.Sum(probs)
	if sum == 0 {
		return nil, fmt.Errorf("split.Random probs sum to 0")
	}
	nr := ix.Len()
	ns := make([]int, np)
	cum := 0
	fnr := float64(nr)
	for i, p := range probs {
		p /= sum
		per := int(math.Round(p * fnr))
		if cum+per > nr {
			per = nr - cum
			if per <= 0 {
				break
			}
		}
		ns[i] = per
		cum += per
	}
	spl := &table.Splits{}
	perm := ix.Clone()
	perm.Permuted()
	cum = 0
	spl.SetLevels("permuted")
	for i, n := range ns {
		nm := ""
		if names != nil {
			nm = names[i]
		} else {
			nm = fmt.Sprintf("p=%v", probs[i]/sum)
		}
		spl.New(ix.Table, []string{nm}, perm.Indexes[cum:cum+n]...)
		cum += n
	}
	return spl, nil
}
