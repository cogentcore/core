// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensormpi

import (
	"fmt"
	"log"

	"cogentcore.org/core/base/mpi"
)

// Alloc allocates n items to current mpi proc based on WorldSize and WorldRank.
// Returns start and end (exclusive) range for current proc.
func AllocN(n int) (st, end int, err error) {
	nproc := mpi.WorldSize()
	if n%nproc != 0 {
		err = fmt.Errorf("tensormpi.AllocN: number: %d is not an even multiple of number of MPI procs: %d -- must be!", n, nproc)
		log.Println(err)
	}
	pt := n / nproc
	st = pt * mpi.WorldRank()
	end = st + pt
	return
}
