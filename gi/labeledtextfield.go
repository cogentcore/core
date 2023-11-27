// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "goki.dev/ki/v2"

// LabeledTextField is a [Label] with optional associated label,
// hint, and error text.
type LabeledTextField struct {
	TextField

	// Label is the label for the text field
	Label string

	// HintText is the hint text for the text field
	HintText string

	// ErrorText is the error text for the text field.
	// If it is specified, it will be shown instead of
	// [LabeledTextField.HintText].
	ErrorText string
}

func (lt *LabeledTextField) ConfigWidget() {
	lt.ConfigParts()
}

func (lt *LabeledTextField) ConfigParts() {
	config := ki.Config{}
	if lt.Label != "" {
		config.Add(LabelType, "label")
	}
	if lt.HintText != "" && lt.ErrorText == "" {
		config.Add(LabelType, "hint")
	}
	if lt.ErrorText != "" {
		config.Add(LabelType, "error")
	}
	lt.ConfigPartsImpl(config)
}
