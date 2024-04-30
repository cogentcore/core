// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package pages

import (
	"net/url"
	"strings"
	"syscall/js"

	"cogentcore.org/core/base/errors"
)

func init() {
	getWebURL = func() string {
		full, err := url.Parse(js.Global().Get("location").Get("href").String())
		if errors.Log(err) != nil {
			return "/"
		}
		basePath, err := url.Parse(js.Global().Get("document").Get("documentElement").Get("dataset").Get("basePath").String())
		if errors.Log(err) != nil {
			return "/"
		}
		base := full.ResolveReference(basePath)
		result := strings.TrimPrefix(full.String(), base.String())
		return "/" + result
	}
}
