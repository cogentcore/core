// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Google authenticates the user with Google using [Auth] and the given configuration
// information and returns the resulting oauth token and user info. It sets the values
// of [AuthConfig.ProviderName], [AuthConfig.ProviderURL], [AuthConfig.ClientID], and
// [AuthConfig.ClientSecret] if they are not already set.
func Google(c *AuthConfig) (*oauth2.Token, *oidc.UserInfo, error) {
	if c.ProviderName == "" {
		c.ProviderName = "google"
	}
	if c.ProviderURL == "" {
		c.ProviderURL = "https://accounts.google.com"
	}
	if c.ClientID == "" {
		c.ClientID = os.Getenv("GOOGLE_OAUTH2_CLIENT_ID")
	}
	if c.ClientSecret == "" {
		c.ClientSecret = os.Getenv("GOOGLE_OAUTH2_CLIENT_SECRET")
	}
	return Auth(c)
}
