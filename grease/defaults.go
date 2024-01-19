// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import "cogentcore.org/core/laser"

// SetFromDefaults sets the values of the given config object
// from `default:` field tag values. Parsing errors are automatically logged.
func SetFromDefaults(cfg any) error {
	return laser.SetFromDefaultTags(cfg)
}
