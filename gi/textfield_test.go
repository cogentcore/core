// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestTextField(t *testing.T) {
	for _, typ := range TextFieldTypesValues() {
		for _, str := range testStrings {
			for _, lic := range testIcons {
				for _, tic := range testIcons1 {
					b := NewBody()
					NewTextField(b).SetType(typ).SetText(str).SetLeadingIcon(lic).SetTrailingIcon(tic)
					nm := testName("textfield", typ, str, lic, tic)
					b.AssertRender(t, nm)
				}
			}
		}
	}
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
	b.AssertRender(t, filepath.Join("textfield", "validator_valid"), func() {
		tf.SendChange() // trigger validation
	})
}

func TestTextFieldValidatorInvalid(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).SetText("my password")
	tf.SetValidator(func() error {
		if len(tf.Text()) < 12 {
			return errors.New("password must be at least 12 characters")
		}
		return nil
	})
	b.AssertRender(t, filepath.Join("textfield", "validator_invalid"), func() {
		tf.SendChange() // trigger validation
	})
}
