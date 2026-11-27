// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"os"
	"slices"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Google authenticates the user with Google using [Auth] and the given configuration
// information and returns the resulting oauth token and user info. It sets the values
// of [AuthConfig.ProviderName], [AuthConfig.ProviderURL], [AuthConfig.ClientID], and
// [AuthConfig.ClientSecret] if they are not already set.
func Google(c *AuthConfig) (*oauth2.Token, *oidc.UserInfo, error) {
	return Auth(googleConfig(c))
}

// GoogleTokenSource authenticates the user with Google using [TokenSource],
// setting the values of the configuration like [Google].
func GoogleTokenSource(c *AuthConfig) (oauth2.TokenSource, *oidc.UserInfo, error) {
	return TokenSource(googleConfig(c))
}

// googleConfig sets the Google values of the given configuration,
// if they are not already set, and returns it.
func googleConfig(c *AuthConfig) *AuthConfig {
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
	return c
}

// Microsoft authenticates the user with Microsoft (a work or school
// account of Microsoft 365, or a personal one, as of Outlook.com) using
// [Auth], setting the values of the configuration like [Google], with the
// client ID of the environment variable MICROSOFT_OAUTH2_CLIENT_ID (of an
// app registered in the Microsoft Entra admin center, as a public client,
// of the platform "Mobile and desktop applications", with the redirect URI
// http://127.0.0.1:5556/auth/microsoft/callback), and no client secret.
func Microsoft(c *AuthConfig) (*oauth2.Token, *oidc.UserInfo, error) {
	return Auth(microsoftConfig(c))
}

// MicrosoftTokenSource authenticates the user with Microsoft using
// [TokenSource], setting the values of the configuration like [Microsoft].
func MicrosoftTokenSource(c *AuthConfig) (oauth2.TokenSource, *oidc.UserInfo, error) {
	return TokenSource(microsoftConfig(c))
}

// microsoftConfig sets the Microsoft values of the given configuration,
// if they are not already set, and returns it: the endpoint for the
// accounts of all organizations, and personal ones, whose issuer is that
// of the organization of the account, the email address from the ID
// token, the offline_access scope for a refresh token, and one prompt.
func microsoftConfig(c *AuthConfig) *AuthConfig {
	if c.ProviderName == "" {
		c.ProviderName = "microsoft"
	}
	if c.ProviderURL == "" {
		c.ProviderURL = "https://login.microsoftonline.com/common/v2.0"
		c.IssuerURL = "https://login.microsoftonline.com/{tenantid}/v2.0"
	}
	if c.ClientID == "" {
		c.ClientID = os.Getenv("MICROSOFT_OAUTH2_CLIENT_ID")
	}
	if !slices.Contains(c.Scopes, "offline_access") {
		c.Scopes = append(slices.Clone(c.Scopes), "offline_access")
	}
	c.UserFromIDToken = true
	c.OnePrompt = true
	return c
}
