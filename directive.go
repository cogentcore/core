// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"reflect"
)

// SetFromDirective sets Config values from a comment directive,
// based on the field names in the Config struct.
// Returns any args that did not start with a `-` flag indicator.
// For more robust error processing, it is assumed that all flagged args (-)
// must refer to fields in the config, so any that fail to match trigger
// an error.  Errors can also result from parsing.
// Errors are automatically logged because these are user-facing.
func SetFromDirective(cfg any, args []string) (nonFlags []string, err error) {
	allArgs := make(map[string]reflect.Value)
	CommandArgs(allArgs) // need these to not trigger not-found errors
	FieldArgNames(cfg, allArgs)
	nonFlags, err = ParseArgs(cfg, args, allArgs, true)
	if err != nil {
		fmt.Println(Usage(cfg))
	}
	return
}
