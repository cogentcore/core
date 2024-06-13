// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"path/filepath"

	"cogentcore.org/core/base/auth"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/views"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func main() {
	b := core.NewBody("Auth scopes and token file example")
	fun := func(token *oauth2.Token, userInfo *oidc.UserInfo) {
		d := core.NewBody()
		core.NewText(d).SetType(core.TextHeadlineMedium).SetText("Basic info")
		views.NewStructView(d).SetStruct(userInfo)
		core.NewText(d).SetType(core.TextHeadlineMedium).SetText("Detailed info")
		claims := map[string]any{}
		errors.Log(userInfo.Claims(&claims))
		views.NewKeyValueTable(d).SetMap(&claims)
		d.AddOKOnly().RunFullDialog(b)
	}
	auth.Buttons(b, &auth.ButtonsConfig{
		SuccessFunc: fun,
		TokenFile: func(provider, email string) string {
			return filepath.Join(core.TheApp.AppDataDir(), provider+"-token.json")
		},
		Scopes: map[string][]string{
			"google": {"https://mail.google.com/"},
		},
	})
	b.RunMainWindow()
}
