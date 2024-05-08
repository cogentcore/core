// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// good reference for impl: https://github.com/helloyi/go-sshclient

// SSHConfig holds ssh configuration and connection info.
// todo: probably could put this in exec, integrate with it.
type SSHConfig struct {
	// user name to connect with
	User string

	// path to ssh keys: ~/.ssh by default
	KeyPath string

	// name of ssh key file in KeyPath -- .pub is appended for public key
	KeyFile string `default:"id_rsa"`

	// host to connect to
	Host string

	// Stdout is the writer to write the standard output of called commands to.
	// It can be set to nil to disable the writing of the standard output.
	Stdout io.Writer

	// Stderr is the writer to write the standard error of called commands to.
	// It can be set to nil to disable the writing of the standard error.
	Stderr io.Writer

	// Stdin is the reader to use as the standard input.
	Stdin io.Reader

	// client is present (non-nil) after a successful Connect
	Client *ssh.Client

	// keeps track of sessions that are being waited upon
	Sessions map[string]*ssh.Session
}

func (sh *SSHConfig) Defaults() {
	sh.KeyFile = "id_rsa"
	usr, err := user.Current()
	if err == nil {
		sh.User = usr.Username
		sh.KeyPath = filepath.Join(usr.HomeDir, ".ssh")
	}
}

// Close terminates any open Sessions and then closes the Client
func (sh *SSHConfig) Close() {
	sh.CloseSessions()
	if sh.Client != nil {
		sh.Client.Close()
	}
	sh.Client = nil
}

// CloseSessions terminates any open Sessions
func (sh *SSHConfig) CloseSessions() {
	if sh.Sessions == nil {
		return
	}
	for _, ses := range sh.Sessions {
		ses.Close()
	}
	sh.Sessions = nil
}

// Connect connects to given host, which can either be just the host
// or user@host. If successful, creates a Client that can be used for
// future sessions.  Otherwise, returns error.  Updates Host (and User)
// fields in the config, for future use.
func (sh *SSHConfig) Connect(host string) error {
	if sh.KeyPath == "" {
		err := fmt.Errorf("ssh: key path (%q) is empty -- must be set", sh.KeyPath)
		return err
	}
	fn := filepath.Join(sh.KeyPath, sh.KeyFile)
	key, err := os.ReadFile(fn)
	if err != nil {
		err = fmt.Errorf("ssh: unable to read private key from: %q %v", fn, err)
		return err
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		err = fmt.Errorf("ssh: unable to parse private key from: %q %v", fn, err)
		return err
	}

	// more info: https://gist.github.com/Skarlso/34321a230cf0245018288686c9e70b2d
	hostKeyCallback, err := knownhosts.New(filepath.Join(sh.KeyPath, "known_hosts"))
	if err != nil {
		log.Fatal("ssh: could not create hostkeycallback function: ", err)
	}

	atidx := strings.Index(host, "@")
	if atidx > 0 {
		sh.User = host[:atidx]
		sh.Host = host[atidx+1:]
		host = sh.Host
	} else {
		sh.Host = host
	}

	config := &ssh.ClientConfig{
		User: sh.User,
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
	}

	// Connect to the remote server and perform the SSH handshake.
	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		err = fmt.Errorf("ssh: unable to connect to %s as user %s: %v", host, sh.User, err)
		return err
	}

	sh.Sessions = make(map[string]*ssh.Session)
	sh.Client = client
	return nil
}

// NewSession creates a new session, sets its input / outputs based on
// config.  Only works if Client already connected.
func (sh *SSHConfig) NewSession() (*ssh.Session, error) {
	if sh.Client == nil {
		return nil, fmt.Errorf("ssh: no client")
	}
	ses, err := sh.Client.NewSession()
	if err != nil {
		return nil, err
	}
	ses.Stdin = sh.Stdin
	ses.Stdout = sh.Stdout
	ses.Stderr = sh.Stderr
	return ses, nil
}

// WaitSession adds the session to list of open sessions, and calls Wait on it.
// it should be called in a goroutine, and will only return when the command
// is completed or terminated.  The given name is used to save the session
// in a map, for later reference.
func (sh *SSHConfig) WaitSession(name string, ses *ssh.Session) error {
	sh.Sessions[name] = ses
	return ses.Wait()
}

// Run runs given command, using config input / outputs.
// Must have already made a successful Connect.
func (sh *SSHConfig) Run(cmd string) error {
	ses, err := sh.NewSession()
	if err != nil {
		return err
	}
	err = ses.Run(cmd)
	ses.Close()
	return err
}
