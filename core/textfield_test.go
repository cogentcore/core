// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"errors"
	"testing"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

func TestTextField(t *testing.T) {
	b := NewBody()
	NewTextField(b)
	b.AssertRender(t, "text-field/basic")
}

func TestTextFieldValidatorValid(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).SetText("my secure password")
	tf.SetValidator(func() error {
		if len(tf.Text()) < 12 {
			return errors.New("password must be at least 12 characters")
		}
		return nil
	})
	b.AssertRender(t, "text-field/validator-valid", func() {
		tf.SendChange() // trigger validation
	})
}

func TestTextFieldValidatorInvalid(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Min.Set(units.Em(15), units.Em(10))
	})
	tf := NewTextField(b).SetText("my password")
	tf.SetValidator(func() error {
		if len(tf.Text()) < 12 {
			return errors.New("password must be at least 12 characters")
		}
		return nil
	})
	b.AssertRenderScreen(t, "text-field/validator-invalid", func() {
		tf.SendChange() // trigger validation
	})
}
