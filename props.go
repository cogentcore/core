// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"github.com/rcoreilly/goki/ki/kit"
)

// Props is the type used for holding generic properties -- the actual Go type
// is a mouthful and not very gui-friendly, and we need some special json methods
type Props map[string]interface{}

var KiT_Props = kit.Types.AddType(&Props{}, PropsProps)

var PropsProps = Props{
	"basic-type": true, // registers props as a basic type avail for type selection in creating property values -- many cases call for nested properties
}

// SubProps returns a value that contains another props, or nil if it doesn't
// exist or isn't a Props
func SubProps(pr map[string]interface{}, key string) Props {
	sp, ok := pr[key]
	if !ok {
		return nil
	}
	spp, ok := sp.(Props)
	if ok {
		return spp
	}
	return nil

}
