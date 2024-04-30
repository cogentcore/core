// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package pages

import (
	"fmt"
	"net/url"
	"strings"
	"syscall/js"

	"cogentcore.org/core/base/errors"
)

func init() {
	getWebURL = func() string {
		full, base, err := getURL()
		if errors.Log(err) != nil {
			return "/"
		}
		result := strings.TrimPrefix(full.String(), base.String())
		return "/" + result
	}
	saveWebURL = func(u string) {
		_, base, err := getURL()
		if errors.Log(err) != nil {
			return
		}
		new, err := url.Parse(u)
		if errors.Log(err) != nil {
			return
		}
		fullNew := base.ResolveReference(new)
		fmt.Println(fullNew)
	}
}

// getURL returns the full current URL and website base URL.
func getURL() (full, base *url.URL, err error) {
	full, err = url.Parse(js.Global().Get("location").Get("href").String())
	if err != nil {
		return
	}
	basePath, err := url.Parse(js.Global().Get("document").Get("documentElement").Get("dataset").Get("basePath").String())
	if err != nil {
		return
	}
	base = full.ResolveReference(basePath)
	return
}
