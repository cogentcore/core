// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package languages

import (
	"fmt"

	"cogentcore.org/core/base/fileinfo"
)

var ParserBytes map[fileinfo.Known][]byte = make(map[fileinfo.Known][]byte)

func OpenParser(sl fileinfo.Known) ([]byte, error) {
	parserBytes, ok := ParserBytes[sl]
	if !ok {
		return nil, fmt.Errorf("langs.OpenParser: no parser bytes for %v", sl)
	}
	return parserBytes, nil
}
