// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"os/user"
	"path/filepath"

	"cogentcore.org/core/base/exec"
)

// User holds user-specific ssh connection configuration settings,
// including Key info.
type User struct {
	// user name to connect with
	User string

	// path to ssh keys: ~/.ssh by default
	KeyPath string

	// name of the private ssh key file in KeyPath. If it does not exist,
	// the other standard ones are used: id_ed25519, id_ecdsa or id_rsa.
	KeyFile string `default:"id_rsa"`
}

func (us *User) Defaults() {
	us.KeyFile = "id_rsa"
	usr, err := user.Current()
	if err == nil {
		us.User = usr.Username
		us.KeyPath = filepath.Join(usr.HomeDir, ".ssh")
	}
}

// Config contains the configuration information that controls
// the behavior of ssh connections and commands. It is used
// to establish a Client connection to a remote host.
// It builds on the shared settings in [exec.Config]
type Config struct {
	exec.Config

	// user name and ssh key info
	User User

	// host name / ip address to connect to. can be blank, in which
	// case it must be specified in the Client.Connect call.
	Host string

	// home directory of user on remote host,
	// which is captured at initial connection time.
	HomeDir string
}

// NewConfig returns a new ssh Config based on given
// [exec.Config] parameters.
func NewConfig(cfg *exec.Config) *Config {
	c := &Config{Config: *cfg}
	c.User.Defaults()
	c.Dir = "" // start empty until we get homedir
	return c
}
