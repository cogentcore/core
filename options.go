// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

// Options contains the options passed to Grease
// that control its behavior.
type Options struct {
	// AppName is the internal name of the Grease app
	// (typically in kebab-case) (see also [AppTitle])
	AppName string

	// AppTitle is the user-visible name of the Grease app
	// (typically in Title Case) (see also [AppName])
	AppTitle string

	// AppAbout is the description of the Grease app
	AppAbout string

	// Fatal is whether to, if there is an error in [Run],
	// print it and fatally exit the program through [os.Exit]
	// with an exit code of 1.
	Fatal bool

	// PrintSuccess is whether to print a message indicating
	// that a command was successful after it is run.
	PrintSuccess bool

	// DefaultEncoding is the default encoding format for config files.
	// currently toml is the only supported format, but others could be added
	// if needed.
	DefaultEncoding string

	// DefaultFiles are the default configuration file paths
	DefaultFiles []string

	// IncludePaths is a list of file paths to try for finding config files
	// specified in Include field or via the command line --config --cfg or -c args.
	// Set this prior to calling Config -- default is current directory '.' and 'configs'
	IncludePaths []string

	// SearchUp indicates whether to search up the filesystem
	// for the default config file by checking the provided default
	// config file location relative to each directory up the tree
	SearchUp bool

	// NeedConfigFile indicates whether a configuration file
	// must be provided for the command to run
	NeedConfigFile bool
}

// DefaultOptions returns a new [Options] value
// with standard default values, based on the given
// app name, app title, and app about.
func DefaultOptions(appName, appTitle, appAbout string) *Options {
	return &Options{
		AppName:         appName,
		AppTitle:        appTitle,
		AppAbout:        appAbout,
		Fatal:           true,
		PrintSuccess:    true,
		DefaultEncoding: "toml",
		DefaultFiles:    []string{"config.toml"},
		IncludePaths:    []string{".", "configs"},
	}
}
