// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/gi"
)

func TestValidatorValid(t *testing.T) {
	b := gi.NewBody()
	v := NewValue(b, validator("my@string"))
	b.AssertRender(t, filepath.Join("text", "validator_valid"), func() {
		v.AsWidgetBase().SendChange() // trigger validation
	})
}

func TestValidatorInvalid(t *testing.T) {
	b := gi.NewBody()
	v := NewValue(b, validator("my string"))
	b.AssertRender(t, filepath.Join("text", "validator_invalid"), func() {
		v.AsWidgetBase().SendChange() // trigger validation
	})
}

type validator string

func (v *validator) Validate() error {
	if !strings.Contains(string(*v), "@") {
		return errors.New("must have an @")
	}
	return nil
}
