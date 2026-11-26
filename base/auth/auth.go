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
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// AuthConfig is the configuration information passed to [Auth].
type AuthConfig struct {
	// Ctx is the context to use. It is [context.TODO] if unspecified.
	// Canceling it stops waiting for the user to sign in in the browser.
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

	// TokenFile is an optional function that returns the filename at which the token for the given user will be stored as JSON,
	// or rather the name of the token in the system keychain, where it is stored if there is one (see [SaveToken]).
	// If it is nil or it returns "", the token is not stored. Also, if it is non-nil, Auth skips the user-facing authentication
	// step if it finds a valid token at the file (ie: remember me). It checks all [AuthConfig.Accounts] until it finds one
	// that works for that step. If [AuthConfig.Accounts] is nil, it checks with a blank ("") email account.
	// The token is saved again whenever its access token is refreshed (see [TokenSource]).
	TokenFile func(email string) string

	// Accounts are optional accounts to check for the remember me feature described in [AuthConfig.TokenFile].
	// If it is nil and TokenFile is not, it defaults to contain one blank ("") element.
	Accounts []string

	// Scopes are additional scopes to request beyond the default "openid", "profile", and "email" scopes
	Scopes []string

	// SelectAccount is whether to always have the user sign in in the
	// browser, choosing the account there, without using a saved token,
	// as for adding another account.
	SelectAccount bool

	// LoginHint, if set, is the email address of the account to suggest
	// when the user signs in in the browser, as for signing in to an
	// account again.
	LoginHint string

	// NoPrompt is whether to only use a saved token, returning
	// [ErrNoToken] if there is none that works, instead of having the
	// user sign in in the browser.
	NoPrompt bool
}

// ScopesError is the error of [Auth] and [TokenSource] when the user did
// not allow all of the access of the requested scopes, as the provider can
// let the user uncheck some of them when signing in.
type ScopesError struct {
	// Missing are the requested scopes that were not allowed.
	Missing []string
}

func (e *ScopesError) Error() string {
	return fmt.Sprintf("not all of the access was allowed when signing in (missing %s): sign in again and allow all of it", strings.Join(e.Missing, ", "))
}

// ErrNoToken is the error of [Auth] and [TokenSource] with
// [AuthConfig.NoPrompt] when there is no saved token that works.
var ErrNoToken = errors.New("auth: no saved token")

// callbackAddress is the address of the local server that gets the
// result of the user signing in in the browser.
const callbackAddress = "127.0.0.1:5556"

// signInMu makes only one sign-in in the browser happen at a time,
// as they use the same local server address.
var signInMu sync.Mutex

// Auth authenticates the user using the given configuration information and returns the
// resulting oauth token and user info. See [AuthConfig] for more information on the
// configuration options. Use [TokenSource] to keep the access token fresh.
func Auth(c *AuthConfig) (*oauth2.Token, *oidc.UserInfo, error) {
	ts, userInfo, err := TokenSource(c)
	if err != nil {
		return nil, nil, err
	}
	token, err := ts.Token()
	if err != nil {
		return nil, nil, err
	}
	return token, userInfo, nil
}

// TokenSource authenticates the user like [Auth], and returns a token
// source whose tokens have a valid access token, refreshed as needed
// (which requires a refresh token, which the provider returns for the
// offline access requested here), and the user info. Each refreshed
// token is saved to the token file (see [AuthConfig.TokenFile]).
func TokenSource(c *AuthConfig) (oauth2.TokenSource, *oidc.UserInfo, error) {
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

	config := &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  "http://" + callbackAddress + "/auth/" + c.ProviderName + "/callback",
		Endpoint:     provider.Endpoint(),
		Scopes:       append([]string{oidc.ScopeOpenID, "profile", "email"}, c.Scopes...),
	}

	// remember me: the first saved token that still works
	if c.TokenFile != nil && !c.SelectAccount {
		accounts := c.Accounts
		if accounts == nil {
			accounts = []string{""}
		}
		for _, account := range accounts {
			tf := c.TokenFile(account)
			if tf == "" {
				continue
			}
			token, err := loadToken(tf)
			if token == nil && err == nil {
				slog.Debug("auth: no saved token", "account", account)
				continue
			}
			slog.Debug("auth: trying the saved token", "account", account, "file", tf,
				"hasRefreshToken", token.RefreshToken != "", "expiry", token.Expiry)
			if err != nil {
				return nil, nil, err
			}
			if token.RefreshToken != "" {
				// refresh now, which checks that the refresh token still
				// works, and gets the scopes that it allows
				token.Expiry = time.Unix(1, 0)
			}
			ts, userInfo, err := tokenSource(c, provider, config, token)
			if err != nil {
				var re *oauth2.RetrieveError
				var se *ScopesError
				if !errors.As(err, &re) && !errors.As(err, &se) {
					// not known not to work, as when offline
					slog.Debug("auth: could not check the saved token", "account", account, "err", err)
					return nil, nil, err
				}
				// such as a revoked or expired refresh token
				slog.Info("auth: saved token does not work", "account", account, "err", err)
				continue
			}
			slog.Debug("auth: the saved token works", "account", account, "email", userInfo.Email)
			return ts, userInfo, nil
		}
	}
	if c.NoPrompt {
		return nil, nil, ErrNoToken
	}

	token, err := signIn(c, config)
	if err != nil {
		slog.Debug("auth: signing in failed", "err", err)
		return nil, nil, err
	}
	ts, userInfo, err := tokenSource(c, provider, config, token)
	if err != nil {
		slog.Debug("auth: the token of signing in does not work", "err", err)
		return nil, nil, err
	}
	slog.Debug("auth: signed in", "email", userInfo.Email)
	return ts, userInfo, nil
}

// tokenSource returns the saving token source for the given token and
// the user info, saving the token.
func tokenSource(c *AuthConfig, provider *oidc.Provider, config *oauth2.Config, token *oauth2.Token) (oauth2.TokenSource, *oidc.UserInfo, error) {
	src := config.TokenSource(c.Ctx, token)
	fresh, err := src.Token()
	if err != nil {
		slog.Debug("auth: getting a fresh token failed", "err", err)
		return nil, nil, err
	}
	slog.Debug("auth: fresh token", "grantedScopes", fresh.Extra("scope"), "expiry", fresh.Expiry,
		"hasRefreshToken", fresh.RefreshToken != "", "refreshed", fresh.AccessToken != token.AccessToken)
	if err := checkScopes(fresh, c.Scopes); err != nil {
		slog.Debug("auth: missing scopes", "err", err)
		return nil, nil, err
	}
	userInfo, err := provider.UserInfo(c.Ctx, oauth2.StaticTokenSource(fresh))
	if err != nil {
		slog.Debug("auth: getting the user info failed", "err", err)
		return nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}
	ts := &savingTokenSource{src: src}
	if c.TokenFile != nil {
		ts.file = c.TokenFile(userInfo.Email)
	}
	if err := ts.save(fresh); err != nil {
		return nil, nil, err
	}
	return ts, userInfo, nil
}

// checkScopes returns a [*ScopesError] if the given token, from the
// provider, does not allow all of the given scopes. Tokens that do not say
// which scopes they allow (only those just received do) are not checked.
func checkScopes(token *oauth2.Token, scopes []string) error {
	granted, _ := token.Extra("scope").(string)
	if granted == "" {
		return nil
	}
	have := strings.Fields(granted)
	var missing []string
	for _, s := range scopes {
		if !slices.Contains(have, s) {
			missing = append(missing, s)
		}
	}
	if len(missing) > 0 {
		return &ScopesError{Missing: missing}
	}
	return nil
}

// signIn has the user sign in in the browser, returning the token.
func signIn(c *AuthConfig, config *oauth2.Config) (*oauth2.Token, error) {
	signInMu.Lock()
	defer signInMu.Unlock()

	b := make([]byte, 16)
	rand.Read(b)
	state := base64.RawURLEncoding.EncodeToString(b)

	ln, err := net.Listen("tcp", callbackAddress)
	if err != nil {
		return nil, fmt.Errorf("starting the local server for signing in: %w", err)
	}
	code := make(chan string, 1)
	sm := http.NewServeMux()
	sm.HandleFunc("/auth/"+c.ProviderName+"/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}
		if e := r.URL.Query().Get("error"); e != "" {
			slog.Debug("auth: the provider returned an error", "error", e, "description", r.URL.Query().Get("error_description"))
			http.Error(w, "Not signed in: "+e, http.StatusBadRequest)
			return
		}
		slog.Debug("auth: got the result of signing in", "scope", r.URL.Query().Get("scope"))
		select {
		case code <- r.URL.Query().Get("code"):
		default:
		}
		w.Write([]byte("<h1>Signed in</h1><p>You can return to the app</p>"))
	})
	srv := &http.Server{Handler: sm}
	go srv.Serve(ln)
	defer srv.Close()

	// offline access for a refresh token; consent so that the provider
	// returns one even if the user has already allowed the app
	prompt := "consent"
	if c.SelectAccount {
		prompt = "select_account consent"
	}
	opts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", prompt)}
	if c.LoginHint != "" {
		opts = append(opts, oauth2.SetAuthURLParam("login_hint", c.LoginHint))
	}
	slog.Debug("auth: opening the browser to sign in", "provider", c.ProviderName, "scopes", config.Scopes,
		"prompt", prompt, "loginHint", c.LoginHint)
	core.TheApp.OpenURL(config.AuthCodeURL(state, opts...))

	var cs string
	select {
	case cs = <-code:
	case <-c.Ctx.Done():
		return nil, c.Ctx.Err()
	}
	token, err := config.Exchange(c.Ctx, cs)
	if err != nil {
		slog.Debug("auth: exchanging the code for a token failed", "err", err)
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}
	slog.Debug("auth: got a token", "grantedScopes", token.Extra("scope"), "hasRefreshToken", token.RefreshToken != "")
	return token, nil
}

// savingTokenSource is a token source that saves each new token
// to its file, if any.
type savingTokenSource struct {
	src  oauth2.TokenSource
	file string

	mu     sync.Mutex
	access string // the access token last saved
}

func (ts *savingTokenSource) Token() (*oauth2.Token, error) {
	token, err := ts.src.Token()
	if err != nil {
		return nil, err
	}
	return token, ts.save(token)
}

// save saves the given token if its access token is new.
func (ts *savingTokenSource) save(token *oauth2.Token) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.file == "" || token.AccessToken == ts.access {
		return nil
	}
	if err := SaveToken(token, ts.file); err != nil {
		slog.Debug("auth: saving the token failed", "file", ts.file, "err", err)
		return err
	}
	slog.Debug("auth: saved the token", "file", ts.file, "expiry", token.Expiry)
	ts.access = token.AccessToken
	return nil
}

// keyringService is the service of the tokens in the system keychain
// (as the macOS Keychain), whose names are their token files.
const keyringService = "Cogent Core auth"

// loadToken returns the token saved for the given token file (see
// [AuthConfig.TokenFile]): from the system keychain, or else from the
// file, which is then moved to the keychain, if there is one. It returns
// nil if there is no saved token.
func loadToken(file string) (*oauth2.Token, error) {
	data, err := keyring.Get(keyringService, file)
	if err == nil {
		token := &oauth2.Token{}
		if err := json.Unmarshal([]byte(data), token); err != nil {
			return nil, fmt.Errorf("reading the saved token: %w", err)
		}
		return token, nil
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		slog.Debug("auth: the keychain is not available", "err", err)
	}
	var token *oauth2.Token
	err = jsonx.Open(&token, file)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && token == nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := SaveToken(token, file); err != nil { // to the keychain
		slog.Warn("auth: could not move the saved token", "file", file, "err", err)
	}
	return token, nil
}

// SaveToken saves the given token for the given token file (see
// [AuthConfig.TokenFile]) in the system keychain (as the macOS Keychain),
// with the name of the file, removing the file if any. Where there is no
// keychain, as on a Linux server, it is saved in the file instead,
// readable only by the user.
func SaveToken(token *oauth2.Token, file string) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	kerr := keyring.Set(keyringService, file, string(data))
	if kerr == nil {
		if err := os.Remove(file); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	slog.Debug("auth: saving the token in a file, without a keychain", "file", file, "err", kerr)
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o600)
}

// DeleteToken deletes the token saved for the given token file (see
// [AuthConfig.TokenFile]), from the system keychain and the file.
func DeleteToken(file string) error {
	err := keyring.Delete(keyringService, file)
	if errors.Is(err, keyring.ErrNotFound) {
		err = nil
	}
	if ferr := os.Remove(file); ferr != nil && !errors.Is(ferr, fs.ErrNotExist) {
		return ferr
	}
	if err != nil {
		slog.Debug("auth: deleting the token from the keychain", "file", file, "err", err)
	}
	return nil
}
