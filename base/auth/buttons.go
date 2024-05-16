// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"context"
	"embed"
	"io/fs"

	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"github.com/coreos/go-oidc/v3/oidc"

	"golang.org/x/oauth2"
)

//go:embed icons/*.svg
var providerIcons embed.FS

func init() {
	icons.AddFS(errors.Log1(fs.Sub(providerIcons, "icons")))
}

// ButtonsConfig is the configuration information passed to [Buttons].
type ButtonsConfig struct {
	// SuccessFunc, if non-nil, is the function called after the user successfully
	// authenticates. It is passed the user's authentication token and info.
	SuccessFunc func(token *oauth2.Token, userInfo *oidc.UserInfo)

	// TokenFile, if non-nil, is the function used to determine what token file function is
	// used for [AuthConfig.TokenFile]. It is passed the provider being used (eg: "google") and the
	// email address of the user authenticating.
	TokenFile func(provider, email string) string

	// Accounts are optional accounts to check for the remember me feature described in [AuthConfig.TokenFile].
	// See [AuthConfig.Accounts] for more information. If it is nil and TokenFile is not, it defaults to contain
	// one blank ("") element.
	Accounts []string

	// Scopes, if non-nil, is a map of scopes to pass to [Auth], keyed by the
	// provider being used (eg: "google").
	Scopes map[string][]string
}

// Buttons adds a new vertical layout to the given parent with authentication
// buttons for major platforms, using the given configuration options. See
// [ButtonsConfig] for more information on the configuration options. The
// configuration options can be nil, in which case default values will be used.
func Buttons(par core.Widget, c *ButtonsConfig) *core.Frame {
	ly := core.NewLayout(par)
	ly.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	GoogleButton(ly, c)
	return ly
}

// Button makes a new button for signing in with the provider
// that has the given name and auth func. It should not typically
// be used by end users; instead, use [Buttons] or the platform-specific
// functions (eg: [Google]). The configuration options can be nil, in
// which case default values will be used.
func Button(par core.Widget, c *ButtonsConfig, provider string, authFunc func(c *AuthConfig) (*oauth2.Token, *oidc.UserInfo, error)) *core.Button {
	if c == nil {
		c = &ButtonsConfig{}
	}
	if c.SuccessFunc == nil {
		c.SuccessFunc = func(token *oauth2.Token, userInfo *oidc.UserInfo) {}
	}
	if c.Scopes == nil {
		c.Scopes = map[string][]string{}
	}

	bt := core.NewButton(par).SetText("Sign in")

	tf := func(email string) string {
		if c.TokenFile != nil {
			return c.TokenFile(provider, email)
		}
		return ""
	}
	ac := &AuthConfig{
		Ctx:          context.TODO(),
		ProviderName: provider,
		TokenFile:    tf,
		Accounts:     c.Accounts,
		Scopes:       c.Scopes[provider],
	}

	auth := func() {
		token, userInfo, err := authFunc(ac)
		if err != nil {
			core.ErrorDialog(bt, err, "Error signing in with "+strcase.ToSentence(provider))
			return
		}
		c.SuccessFunc(token, userInfo)
	}
	bt.OnClick(func(e events.Event) {
		auth()
	})

	// if we have a valid token file, we auth immediately without the user clicking on the button
	if c.TokenFile != nil {
		if c.Accounts == nil {
			c.Accounts = []string{""}
		}
		for _, account := range c.Accounts {
			tf := c.TokenFile(provider, account)
			if tf != "" {
				exists, err := dirs.FileExists(tf)
				if err != nil {
					core.ErrorDialog(bt, err, "Error searching for saved "+strcase.ToSentence(provider)+" auth token file")
					return bt
				}
				if exists {
					// have to wait until the scene is shown in case any dialogs are created
					bt.OnShow(func(e events.Event) {
						auth()
					})
				}
			}
		}
	}
	return bt
}

// GoogleButton adds a new button for signing in with Google
// to the given parent using the given configuration information.
func GoogleButton(par core.Widget, c *ButtonsConfig) *core.Button {
	bt := Button(par, c, "google", Google).SetType(core.ButtonOutlined).
		SetText("Sign in with Google").SetIcon("sign-in-with-google")
	bt.Style(func(s *styles.Style) {
		s.Color = colors.C(colors.Scheme.OnSurface)
	})
	return bt
}
