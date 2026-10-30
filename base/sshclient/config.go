// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"os/user"
	"path/filepath"
	"time"

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

	// KeepAlive is the interval between keepalive requests to the host,
	// which detect a lost connection, as when the network is lost
	// (see [Client]). 0 turns them off.
	KeepAlive time.Duration

	// Timeout is the time to wait for the connection to the host, and for
	// the replies to keepalive requests. 0 is no limit.
	Timeout time.Duration

	// LoginEnv is whether the commands get the environment variables of an
	// interactive login shell of the user on the host, such as PATH, which
	// it gets once when it connects. The commands are otherwise run by a
	// shell that is not interactive, for which a standard ~/.bashrc returns
	// before it sets them.
	LoginEnv bool
}

// NewConfig returns a new ssh Config based on given
// [exec.Config] parameters.
func NewConfig(cfg *exec.Config) *Config {
	c := &Config{Config: *cfg, KeepAlive: 15 * time.Second, Timeout: 15 * time.Second, LoginEnv: true}
	c.User.Defaults()
	c.Dir = "" // start empty until we get homedir
	return c
}
