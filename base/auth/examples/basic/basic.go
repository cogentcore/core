// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/base/auth"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/views"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func main() {
	b := core.NewBody("Auth basic example")
	fun := func(token *oauth2.Token, userInfo *oidc.UserInfo) {
		d := core.NewBody().AddTitle("User info")
		core.NewText(d).SetType(core.TextHeadlineMedium).SetText("Basic info")
		views.NewForm(d).SetStruct(userInfo)
		core.NewText(d).SetType(core.TextHeadlineMedium).SetText("Detailed info")
		claims := map[string]any{}
		errors.Log(userInfo.Claims(&claims))
		views.NewKeyedList(d).SetMap(&claims)
		d.AddOKOnly().RunFullDialog(b)
	}
	auth.Buttons(b, &auth.ButtonsConfig{SuccessFunc: fun})
	b.RunMainWindow()
}
