// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package strcase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTo(t *testing.T) {
	s := "some HTTPServer name"
	want := map[Cases]string{
		LowerCase:      "some httpserver name",
		UpperCase:      "SOME HTTPSERVER NAME",
		SnakeCase:      "some_http_server_name",
		SNAKECase:      "SOME_HTTP_SERVER_NAME",
		KebabCase:      "some-http-server-name",
		KEBABCase:      "SOME-HTTP-SERVER-NAME",
		CamelCase:      "SomeHTTPServerName",
		LowerCamelCase: "someHTTPServerName",
		TitleCase:      "Some HTTP Server Name",
		SentenceCase:   "Some HTTP server name",
	}
	for c, w := range want {
		assert.Equal(t, w, To(s, c), c.String())
	}
	assert.Equal(t, s, To(s, Cases(-1)), "an unknown case is unchanged")
}
