// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package content

// OfflineURL is the non-web base url, which can be set to allow
// docs to refer to this in frontmatter.
var OfflineURL = ""

// just for printing
func (ct *Content) getPrintURL() string { return OfflineURL }
func (ct *Content) getWebURL() string   { return "" }
func (ct *Content) saveWebURL()         {}
func (ct *Content) handleWebPopState()  {}
