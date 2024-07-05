// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate

import "cogentcore.org/core/cmd/core/config"

// Icons does any necessary generation for icons.
func Icons(c *config.Config) error {
	if c.Generate.Icons == "" {
		return nil
	}
	return nil
}
