// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensormpi

import (
	"errors"
	"fmt"
	"math/rand"

	"cogentcore.org/core/base/mpi"
)

// RandCheck checks that the current random numbers generated across each
// MPI processor are identical.
func RandCheck(comm *mpi.Comm) error {
	ws := comm.Size()
	rnd := rand.Int()
	src := []int{rnd}
	agg := make([]int, ws)
	err := comm.AllGatherInt(agg, src)
	if err != nil {
		return err
	}
	errs := ""
	for i := range agg {
		if agg[i] != rnd {
			errs += fmt.Sprintf("%d ", i)
		}
	}
	if errs != "" {
		err = errors.New("tensormpi.RandCheck: random numbers differ in procs: " + errs)
		mpi.Printf("%s\n", err)
	}
	return err
}
