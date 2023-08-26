// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import "github.com/goki/ki/kit"

// SetFromDefaults sets Config values from field tag `def:` values.
// Parsing errors are automatically logged.
func SetFromDefaults(cfg any) error {
	return kit.SetFromDefaultTags(cfg)
}
