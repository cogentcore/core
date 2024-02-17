// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package langs

import (
	"fmt"

	"cogentcore.org/core/fi"
)

var ParserBytes = make(map[fi.Known][]byte)

func OpenParser(sl fi.Known) ([]byte, error) {
	parserBytes, ok := ParserBytes[sl]
	if !ok {
		return nil, fmt.Errorf("langs.OpenParser: no parser bytes for %v", sl)
	}
	return parserBytes, nil
}
