// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/jsonx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// fakeProvider is an OpenID Connect provider for testing, whose
// refresh tokens are "refresh-<email>", and which gives access tokens
// "access-<email>-<n>", valid for an hour.
type fakeProvider struct {
	*httptest.Server
	refreshes atomic.Int32

	// granted, if set, are the scopes that the tokens allow, as the
	// provider says in its responses.
	granted atomic.Value
}

func newFakeProvider(t *testing.T) *fakeProvider {
	p := &fakeProvider{}
	mux := http.NewServeMux()
	p.Server = httptest.NewServer(mux)
	t.Cleanup(p.Close)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 p.URL,
			"authorization_endpoint": p.URL + "/auth",
			"token_endpoint":         p.URL + "/token",
			"userinfo_endpoint":      p.URL + "/userinfo",
			"jwks_uri":               p.URL + "/jwks",
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		rt := r.Form.Get("refresh_token")
		email, ok := strings.CutPrefix(rt, "refresh-")
		if r.Form.Get("grant_type") != "refresh_token" || !ok {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}
		n := p.refreshes.Add(1)
		w.Header().Set("Content-Type", "application/json")
		res := map[string]any{
			"access_token": "access-" + email + "-" + string(rune('0'+n)),
			"token_type":   "Bearer",
			"expires_in":   3600,
		}
		if g, _ := p.granted.Load().(string); g != "" {
			res["scope"] = g
		}
		json.NewEncoder(w).Encode(res)
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		access := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		rest, ok := strings.CutPrefix(access, "access-")
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		email := rest[:strings.LastIndex(rest, "-")]
		json.NewEncoder(w).Encode(map[string]any{"sub": email, "email": email, "email_verified": true})
	})
	return p
}

// saveToken saves a token for the given email, whose access token has expired.
func saveToken(t *testing.T, file, email string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o700))
	require.NoError(t, jsonx.Save(&oauth2.Token{AccessToken: "access-" + email + "-old", TokenType: "Bearer",
		RefreshToken: "refresh-" + email, Expiry: time.Now().Add(-time.Hour)}, file))
}

func TestTokenSource(t *testing.T) {
	p := newFakeProvider(t)
	dir := t.TempDir()
	tokenFile := func(email string) string {
		if email == "" {
			return ""
		}
		return filepath.Join(dir, email, "token.json")
	}
	config := func(accounts ...string) *AuthConfig {
		return &AuthConfig{ProviderName: "fake", ProviderURL: p.URL, ClientID: "id", ClientSecret: "secret",
			TokenFile: tokenFile, Accounts: accounts, NoPrompt: true}
	}

	_, _, err := TokenSource(config("a@x.test"))
	assert.ErrorIs(t, err, ErrNoToken, "no saved token")

	saveToken(t, tokenFile("b@x.test"), "b@x.test")
	ts, userInfo, err := TokenSource(config("a@x.test", "b@x.test"))
	require.NoError(t, err, "the token of the second account")
	assert.Equal(t, "b@x.test", userInfo.Email)
	token, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "access-b@x.test-1", token.AccessToken, "refreshed")

	var saved oauth2.Token
	require.NoError(t, jsonx.Open(&saved, tokenFile("b@x.test")))
	assert.Equal(t, "access-b@x.test-1", saved.AccessToken, "the refreshed token is saved")
	assert.Equal(t, "refresh-b@x.test", saved.RefreshToken, "with the refresh token")

	// a revoked refresh token does not work
	require.NoError(t, jsonx.Save(&oauth2.Token{AccessToken: "x", RefreshToken: "revoked",
		Expiry: time.Now().Add(-time.Hour)}, tokenFile("b@x.test")))
	_, _, err = TokenSource(config("b@x.test"))
	assert.ErrorIs(t, err, ErrNoToken)
}

// TestScopes tests that a saved token that does not allow all of the
// requested scopes, as when the user unchecked one when signing in,
// is not used.
func TestScopes(t *testing.T) {
	p := newFakeProvider(t)
	dir := t.TempDir()
	tokenFile := func(email string) string { return filepath.Join(dir, email, "token.json") }
	mail := "https://mail.google.com/"
	config := &AuthConfig{ProviderName: "fake", ProviderURL: p.URL, ClientID: "id", ClientSecret: "secret",
		TokenFile: tokenFile, Accounts: []string{"c@x.test"}, Scopes: []string{mail}, NoPrompt: true}
	saveToken(t, tokenFile("c@x.test"), "c@x.test")

	p.granted.Store("openid https://www.googleapis.com/auth/userinfo.email")
	_, _, err := TokenSource(config)
	assert.ErrorIs(t, err, ErrNoToken, "without the mail scope")

	p.granted.Store("openid " + mail)
	ts, _, err := TokenSource(config)
	require.NoError(t, err)
	_, err = ts.Token()
	assert.NoError(t, err)

	// a saved token that has not expired is refreshed to check it
	require.NoError(t, jsonx.Save(&oauth2.Token{AccessToken: "access-c@x.test-old", TokenType: "Bearer",
		RefreshToken: "refresh-c@x.test", Expiry: time.Now().Add(time.Hour)}, tokenFile("c@x.test")))
	p.granted.Store("openid")
	_, _, err = TokenSource(config)
	assert.ErrorIs(t, err, ErrNoToken, "checked by refreshing")

	var se *ScopesError
	assert.ErrorAs(t, checkScopes((&oauth2.Token{}).WithExtra(map[string]any{"scope": "openid"}), []string{mail}), &se)
	assert.Equal(t, []string{mail}, se.Missing)
	assert.NoError(t, checkScopes(&oauth2.Token{}, []string{mail}), "unknown scopes are not checked")
}
