// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAcceptLanguage tests the Accept-Language header of POSIX locales.
func TestAcceptLanguage(t *testing.T) {
	assert.Equal(t, "en-US,en;q=0.9", acceptLanguage(""))
	assert.Equal(t, "en-US,en;q=0.9", acceptLanguage("C"))
	assert.Equal(t, "en-US,en;q=0.9", acceptLanguage("en_US.UTF-8"))
	assert.Equal(t, "fr-CA,fr;q=0.9,en;q=0.8", acceptLanguage("fr_CA.UTF-8"))
	assert.Equal(t, "de,en;q=0.8", acceptLanguage("de@euro"))
}

// TestRequestHeaders tests that the requests of pages have the headers of
// browsers, with a user agent of the form of browsers.
func TestRequestHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
	}))
	defer srv.Close()
	resp, err := getURL(srv.URL)
	assert.NoError(t, err)
	resp.Body.Close()
	ua := got.Get("User-Agent")
	assert.True(t, strings.HasPrefix(ua, "Mozilla/5.0 ("), ua)
	assert.Contains(t, ua, "CogentCore/")
	assert.NotContains(t, ua, "compatible;")
	assert.Contains(t, got.Get("Accept"), "text/html")
	assert.NotEmpty(t, got.Get("Accept-Language"))
	assert.Contains(t, got.Get("Accept-Encoding"), "gzip")
}
