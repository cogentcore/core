// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package auth provides a system for identifying and authenticating
// users through third party cloud systems in Cogent Core apps.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// AuthConfig is the configuration information passed to [Auth].
type AuthConfig struct {
	// Ctx is the context to use. It is [context.TODO] if unspecified.
	Ctx context.Context

	// ProviderName is the name of the provider to authenticate with (eg: "google")
	ProviderName string

	// ProviderURL is the URL of the provider (eg: "https://accounts.google.com")
	ProviderURL string

	// ClientID is the client ID for the app, which is typically obtained through a developer oauth
	// portal (eg: the Credentials section of https://console.developers.google.com/).
	ClientID string

	// ClientSecret is the client secret for the app, which is typically obtained through a developer oauth
	// portal (eg: the Credentials section of https://console.developers.google.com/).
	ClientSecret string

	// TokenFile is an optional function that returns the filename at which the token for the given user will be stored as JSON.
	// If it is nil or it returns "", the token is not stored. Also, if it is non-nil, Auth skips the user-facing authentication
	// step if it finds a valid token at the file (ie: remember me). It checks all [AuthConfig.Accounts] until it finds one
	// that works for that step. If [AuthConfig.Accounts] is nil, it checks with a blank ("") email account.
	TokenFile func(email string) string

	// Accounts are optional accounts to check for the remember me feature described in [AuthConfig.TokenFile].
	// If it is nil and TokenFile is not, it defaults to contain one blank ("") element.
	Accounts []string

	// Scopes are additional scopes to request beyond the default "openid", "profile", and "email" scopes
	Scopes []string
}

// Auth authenticates the user using the given configuration information and returns the
// resulting oauth token and user info. See [AuthConfig] for more information on the
// configuration options.
func Auth(c *AuthConfig) (*oauth2.Token, *oidc.UserInfo, error) {
	if c.Ctx == nil {
		c.Ctx = context.TODO()
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		slog.Warn("got empty client id and/or client secret; did you forgot to set env variables?")
	}

	provider, err := oidc.NewProvider(c.Ctx, c.ProviderURL)
	if err != nil {
		return nil, nil, err
	}

	config := oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  "http://127.0.0.1:5556/auth/" + c.ProviderName + "/callback",
		Endpoint:     provider.Endpoint(),
		Scopes:       append([]string{oidc.ScopeOpenID, "profile", "email"}, c.Scopes...),
	}

	var token *oauth2.Token

	if c.TokenFile != nil {
		if c.Accounts == nil {
			c.Accounts = []string{""}
		}
		for _, account := range c.Accounts {
			tf := c.TokenFile(account)
			if tf != "" {
				err := jsonx.Open(&token, tf)
				if err != nil && !errors.Is(err, fs.ErrNotExist) {
					return nil, nil, err
				}
				break
			}
		}
	}

	// if we didn't get it through remember me, we have to get it manually
	if token == nil {
		b := make([]byte, 16)
		rand.Read(b)
		state := base64.RawURLEncoding.EncodeToString(b)

		code := make(chan string)

		sm := http.NewServeMux()
		sm.HandleFunc("/auth/"+c.ProviderName+"/callback", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("state") != state {
				http.Error(w, "state did not match", http.StatusBadRequest)
				return
			}
			code <- r.URL.Query().Get("code")
			w.Write([]byte("<h1>Signed in</h1><p>You can return to the app</p>"))
		})
		// TODO(kai/auth): more graceful closing / error handling
		go http.ListenAndServe("127.0.0.1:5556", sm)

		core.TheApp.OpenURL(config.AuthCodeURL(state))

		cs := <-code

		token, err = config.Exchange(c.Ctx, cs)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to exchange token: %w", err)
		}
	}

	tokenSource := config.TokenSource(c.Ctx, token)
	// the access token could have changed
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, nil, err
	}

	userInfo, err := provider.UserInfo(c.Ctx, tokenSource)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	if c.TokenFile != nil {
		tf := c.TokenFile(userInfo.Email)
		if tf != "" {
			err := os.MkdirAll(filepath.Dir(tf), 0700)
			if err != nil {
				return nil, nil, err
			}
			// TODO(kai/auth): more secure saving of token file
			err = jsonx.Save(token, tf)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	return newToken, userInfo, nil
}
