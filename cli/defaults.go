// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/reflectx"
)

// SetFromDefaults sets the values of the given config object
// from `default:` struct field tag values. Errors are automatically
// logged in addition to being returned.
func SetFromDefaults(cfg any) error {
	return errors.Log(reflectx.SetFromDefaultTags(cfg))
}
