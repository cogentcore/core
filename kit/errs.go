// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"errors"
	"strings"

	"goki.dev/ki/ints"
)

// AllErrors returns an err as a concatenation of errors (nil if none).
// no more than maxN are included (typically 10).
func AllErrors(errs []error, maxN int) error {
	if len(errs) == 0 {
		return nil
	}
	mx := ints.MinInt(maxN, len(errs))
	ers := make([]string, mx)
	for i := 0; i < mx; i++ {
		ers[i] = errs[i].Error()
	}
	return errors.New(strings.Join(ers, "\n"))
}
