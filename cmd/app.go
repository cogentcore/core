// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

// App is the main app type that handles
// the logic for the GoKi tool
type App struct {

	// the name of the app
	Name string `desc:"the name of the app"`

	// the version of the app
	Version string `desc:"the version of the app"`
}

// TheApp is the singular instance of [App]
var TheApp = &App{}
