// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

// Client represents a persistent connection to an ssh host.
// Commands are run by creating [ssh.Session]s from this client.
//
// The connection is checked with a keepalive request every [Config.KeepAlive],
// and closed if the host does not reply within [Config.Timeout], as happens
// when the network is lost, so that commands waiting on it fail instead of
// hanging. A lost connection is made again the next time it is used
// (see [Client.CheckConnection]), as when the network comes back.
type Client struct {
	Config

	// ssh client is present (non-nil) after a successful Connect,
	// and nil while the connection is lost.
	Client *ssh.Client

	// keeps track of sessions that are being waited upon
	Sessions map[string]*ssh.Session

	// sessionCounter increments number of Sessions added
	// over the lifetime of the Client.
	sessionCounter int

	// exports is the shell code that sets the variables of the login
	// environment of the user for each command (see [Config.LoginEnv]).
	exports string

	// sftpClient copies files, made when first needed (see [Client.SFTP]).
	sftpClient sftpConn

	// addr and sshConfig are those of the connection, for reconnecting.
	addr      string
	sshConfig *ssh.ClientConfig

	// lost is closed when the connection is lost.
	lost chan struct{}

	// lastDial is the time of the last attempt to reconnect, which is
	// only tried again after [Config.Timeout].
	lastDial time.Time

	// mu protects the connection while it is checked and made again.
	mu sync.Mutex
}

// NewClient returns a new Client using given [Config] configuration
// parameters.
func NewClient(cfg *Config) *Client {
	cl := &Client{Config: *cfg}
	return cl
}

// Close terminates any open Sessions and then closes
// the Client connection, which is not made again.
func (cl *Client) Close() {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.closeConnection()
	cl.addr = ""
}

// closeConnection closes the sessions, SFTP and ssh client of the
// connection.
func (cl *Client) closeConnection() {
	cl.CloseSessions()
	if cl.sftpClient != nil {
		cl.sftpClient.Close()
		cl.sftpClient = nil
	}
	if cl.Client != nil {
		cl.Client.Close()
	}
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
			return errors.New("ssh: Connect host is empty and no default host set in Config")
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
	// The key file, or the other standard ones if it does not exist.
	fn := filepath.Join(cl.User.KeyPath, cl.User.KeyFile)
	key, err := os.ReadFile(fn)
	for _, kf := range []string{"id_ed25519", "id_ecdsa", "id_rsa"} {
		if !errors.Is(err, fs.ErrNotExist) {
			break
		}
		if k, kerr := os.ReadFile(filepath.Join(cl.User.KeyPath, kf)); kerr == nil {
			key, err = k, nil
		}
	}
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
		return fmt.Errorf("ssh: could not read the known hosts: %w", err)
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
	// The port is 22 unless the host has one, as in host:2222.
	addr := host
	if _, _, err := net.SplitHostPort(host); err != nil {
		addr = net.JoinHostPort(host, "22")
	}
	cl.mu.Lock()
	cl.closeConnection()
	cl.addr, cl.sshConfig = addr, config
	err = cl.dial()
	cl.mu.Unlock()
	if err != nil {
		return err
	}
	cl.GetHomeDir()
	if cl.LoginEnv {
		cl.getLoginEnv()
	}
	return nil
}

// dial makes the connection to the host, with the [Config.Timeout] for
// the connection and the SSH handshake, and starts the keepalive requests.
func (cl *Client) dial() error {
	cl.lastDial = time.Now()
	conn, err := net.DialTimeout("tcp", cl.addr, cl.Timeout)
	if err != nil {
		return fmt.Errorf("ssh: unable to connect to %s as user %s: %v", cl.Host, cl.User.User, err)
	}
	if cl.Timeout > 0 {
		conn.SetDeadline(time.Now().Add(cl.Timeout))
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, cl.addr, cl.sshConfig)
	if err != nil {
		conn.Close()
		return fmt.Errorf("ssh: unable to connect to %s as user %s: %v", cl.Host, cl.User.User, err)
	}
	conn.SetDeadline(time.Time{})
	client := ssh.NewClient(c, chans, reqs)
	cl.Sessions = make(map[string]*ssh.Session)
	cl.Client = client
	lost := make(chan struct{})
	cl.lost = lost
	var once sync.Once
	setLost := func() { once.Do(func() { close(lost) }) }
	go func() {
		client.Wait()
		setLost()
	}()
	if cl.KeepAlive > 0 {
		go cl.keepAlive(client, lost, setLost)
	}
	return nil
}

// keepAlive sends keepalive requests on the given connection every
// [Config.KeepAlive] until it is lost, and closes it when the host does
// not reply within [Config.Timeout], calling setLost first, so that the
// connection is known to be lost when what is waiting on it fails.
func (cl *Client) keepAlive(client *ssh.Client, lost chan struct{}, setLost func()) {
	tick := time.NewTicker(cl.KeepAlive)
	defer tick.Stop()
	for {
		select {
		case <-lost:
			return
		case <-tick.C:
		}
		reply := make(chan error, 1)
		go func() {
			// a host that does not know the request replies false, which is fine
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			reply <- err
		}()
		timeout := time.NewTimer(cl.Timeout)
		select {
		case err := <-reply:
			timeout.Stop()
			if err == nil {
				continue
			}
		case <-timeout.C:
			slog.Error("ssh: lost connection: no reply from host", "host", cl.Host)
		}
		setLost()
		client.Close() // also ends what is waiting on it
		return
	}
}

// CheckConnection returns nil if the client is connected, and otherwise
// tries to connect again if the connection was lost, as when the network
// was lost, returning an error if that fails. It only tries again after
// the [Config.Timeout] since the last try, returning an error until then,
// so that commands fail fast while the host cannot be reached.
// It returns an error if the client was never connected, or was closed.
func (cl *Client) CheckConnection() error {
	_, err := cl.sshClient()
	return err
}

// sshClient returns the ssh client of the connection, connecting again
// if it was lost (see [Client.CheckConnection]).
func (cl *Client) sshClient() (*ssh.Client, error) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if cl.Client != nil {
		select {
		case <-cl.lost:
			cl.closeConnection()
		default:
			return cl.Client, nil
		}
	}
	if cl.addr == "" {
		return nil, errors.New("ssh: no client, must Connect first")
	}
	if time.Since(cl.lastDial) < cl.Timeout {
		return nil, fmt.Errorf("ssh: not connected to %s: the connection was lost", cl.Host)
	}
	if err := cl.dial(); err != nil {
		return nil, err
	}
	slog.Info("ssh: connected again", "host", cl.Host)
	return cl.Client, nil
}

// NewSession creates a new session, sets its input / outputs based on
// config.  Only works if Client already connected, and connects again
// if the connection was lost (see [Client.CheckConnection]).
func (cl *Client) NewSession(interact bool) (*ssh.Session, error) {
	if interact {
		return cl.newSession(nil)
	}
	client, err := cl.sshClient()
	if err != nil {
		return nil, err
	}
	ses, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	ses.Stdout = cl.StdIO.Out
	ses.Stderr = cl.StdIO.Err
	return ses, nil
}

// newSession returns a new interactive session, with a terminal on the
// host (pty), as [Client.NewSession] does. If the given terminal of the
// input is not nil, the terminal on the host has its size and echoes the
// input, which it gets from it while the command runs (see [forwardInput]).
func (cl *Client) newSession(tty *os.File) (*ssh.Session, error) {
	ses, err := cl.NewSession(false)
	if err != nil {
		return nil, err
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	rows, cols := 40, 80
	if tty != nil {
		modes[ssh.ECHO] = 1
		if w, h, err := term.GetSize(int(tty.Fd())); err == nil {
			rows, cols = h, w
		}
	}
	err = ses.RequestPty("xterm", rows, cols, modes)
	if err != nil {
		slog.Error(err.Error())
	}
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
	ses, err := cl.NewSession(false)
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
	cl.HomeDir = strings.TrimSpace(buf.String())
	cl.Dir = cl.HomeDir
	slog.Debug("ssh: home directory", "Host", cl.Host, "Dir", cl.HomeDir)
	return nil
}
