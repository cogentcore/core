// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bramvdbogaerde/go-scp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Client represents a persistent connection to an ssh host.
// Commands are run by creating [ssh.Session]s from this client.
type Client struct {
	Config

	// ssh client is present (non-nil) after a successful Connect
	Client *ssh.Client

	// keeps track of sessions that are being waited upon
	Sessions map[string]*ssh.Session

	// sessionCounter increments number of Sessions added
	// over the lifetime of the Client.
	sessionCounter int

	// scpClient manages scp file copying
	scpClient *scp.Client
}

// NewClient returns a new Client using given [Config] configuration
// parameters.
func NewClient(cfg *Config) *Client {
	cl := &Client{Config: *cfg}
	return cl
}

// Close terminates any open Sessions and then closes
// the Client connection.
func (cl *Client) Close() {
	cl.CloseSessions()
	if cl.Client != nil {
		cl.Client.Close()
	}
	cl.scpClient = nil
	cl.Client = nil
}

// CloseSessions terminates any open Sessions that are
// still Waiting for the associated process to finish.
func (cl *Client) CloseSessions() {
	if cl.Sessions == nil {
		return
	}
	for _, ses := range cl.Sessions {
		ses.Close()
	}
	cl.Sessions = nil
}

// Connect connects to given host, which can either be just the host
// or user@host. If host is empty, the Config default host will be used
// if non-empty, or an error is returned.
// If successful, creates a Client that can be used for
// future sessions.  Otherwise, returns error.
// This updates the Host (and User) fields in the config, for future
// reference.
func (cl *Client) Connect(host string) error {
	if host == "" {
		if cl.Host == "" {
			return fmt.Errorf("ssh: Connect host is empty and no default host set in Config")
		}
		host = cl.Host
	}
	atidx := strings.Index(host, "@")
	if atidx > 0 {
		cl.User.User = host[:atidx]
		cl.Host = host[atidx+1:]
		host = cl.Host
	} else {
		cl.Host = host
	}

	if cl.User.KeyPath == "" {
		return fmt.Errorf("ssh: key path (%q) is empty -- must be set", cl.User.KeyPath)
	}
	fn := filepath.Join(cl.User.KeyPath, cl.User.KeyFile)
	key, err := os.ReadFile(fn)
	if err != nil {
		return fmt.Errorf("ssh: unable to read private key from: %q %v", fn, err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return fmt.Errorf("ssh: unable to parse private key from: %q %v", fn, err)
	}

	// more info: https://gist.github.com/Skarlso/34321a230cf0245018288686c9e70b2d
	hostKeyCallback, err := knownhosts.New(filepath.Join(cl.User.KeyPath, "known_hosts"))
	if err != nil {
		log.Fatal("ssh: could not create hostkeycallback function: ", err)
	}

	config := &ssh.ClientConfig{
		User: cl.User.User,
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
	}

	// Connect to the remote server and perform the SSH handshake.
	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		err = fmt.Errorf("ssh: unable to connect to %s as user %s: %v", host, cl.User, err)
		return err
	}

	cl.Sessions = make(map[string]*ssh.Session)
	cl.Client = client
	cl.GetHomeDir()
	return nil
}

// NewSession creates a new session, sets its input / outputs based on
// config.  Only works if Client already connected.
func (cl *Client) NewSession() (*ssh.Session, error) {
	if cl.Client == nil {
		return nil, fmt.Errorf("ssh: no client, must Connect first")
	}
	ses, err := cl.Client.NewSession()
	if err != nil {
		return nil, err
	}
	ses.Stdin = nil // cl.StdIO.In // todo: cannot set this like this!
	ses.Stdout = cl.StdIO.Out
	ses.Stderr = cl.StdIO.Err
	return ses, nil
}

// WaitSession adds the session to list of open sessions,
// and calls Wait on it.
// It should be called in a goroutine, and will only return
// when the command is completed or terminated.
// The given name is used to save the session
// in a map, for later reference. If left blank,
// the name will be a number that increases with each
// such session created.
func (cl *Client) WaitSession(name string, ses *ssh.Session) error {
	if name == "" {
		name = fmt.Sprintf("%d", cl.sessionCounter)
	}
	cl.Sessions[name] = ses
	cl.sessionCounter++
	return ses.Wait()
}

// GetHomeDir runs "pwd" on the host to get the users home dir,
// called right after connecting.
func (cl *Client) GetHomeDir() error {
	ses, err := cl.NewSession()
	if err != nil {
		return err
	}
	defer ses.Close()
	buf := &bytes.Buffer{}
	ses.Stdout = buf
	err = ses.Run("pwd")
	if err != nil {
		return fmt.Errorf("ssh: unable to get home directory through pwd: %v", err)
	}
	cl.HomeDir = buf.String()
	cl.Dir = cl.HomeDir
	fmt.Println("home directory:", cl.HomeDir)
	return nil
}
