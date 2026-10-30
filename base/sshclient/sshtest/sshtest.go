// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sshtest provides an SSH server for testing code that uses
// [cogentcore.org/core/base/sshclient], running in the test process on
// the local host: it runs commands with sh in its root directory, which
// is the home directory of the remote host, and serves SFTP.
package sshtest

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cogentcore.org/core/text/terminal/pty"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Server is an SSH server for testing.
type Server struct {

	// Addr is the address of the server, as host:port.
	Addr string

	// Root is the root directory of the server, which is its home
	// directory for the commands that it runs.
	Root string

	// KeyPath is a directory with the private key of the client, as
	// id_rsa, and a known_hosts file with the key of the server, for
	// [cogentcore.org/core/base/sshclient.User.KeyPath].
	KeyPath string

	// Pty is whether the commands run in a terminal (pty) on the server
	// when the client asks for one, as on a real server, such that the
	// terminal handles their input, as for Control+C. It is off by
	// default, as a terminal ends the lines of the output with \r\n.
	Pty atomic.Bool

	listener net.Listener

	// down is whether the network is down (see [Server.NetworkDown]).
	down atomic.Bool

	// conns are the connections to the server.
	conns   []*conn
	connsMu sync.Mutex
}

// conn is a connection to the server, which drops all of the data that
// it reads and writes once it is dead, as a connection does when the
// network is lost, without it being closed.
type conn struct {
	net.Conn
	dead atomic.Bool
}

func (c *conn) Read(b []byte) (int, error) {
	for {
		n, err := c.Conn.Read(b)
		if err != nil || !c.dead.Load() {
			return n, err
		}
	}
}

func (c *conn) Write(b []byte) (int, error) {
	if c.dead.Load() {
		return len(b), nil
	}
	return c.Conn.Write(b)
}

// NetworkDown simulates the loss of the network between the clients and
// the server, as when a laptop sleeps or changes networks: the existing
// connections drop all data from then on, without being closed, so a
// client gets no replies, and new connections are closed right away,
// until [Server.NetworkUp].
func (s *Server) NetworkDown() {
	s.down.Store(true)
	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	for _, c := range s.conns {
		c.dead.Store(true)
	}
}

// NetworkUp ends [Server.NetworkDown], such that new connections work.
// The connections from before stay dead, as in a network.
func (s *Server) NetworkUp() {
	s.down.Store(false)
}

// closeConns closes all of the connections.
func (s *Server) closeConns() {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	for _, c := range s.conns {
		c.Conn.Close()
	}
	s.conns = nil
}

// NewServer starts a new SSH server for the test, which it stops at the
// end of the test, with a new temporary root directory.
func NewServer(t testing.TB) *Server {
	t.Helper()
	s := &Server{Root: t.TempDir(), KeyPath: t.TempDir()}
	if root, err := filepath.EvalSymlinks(s.Root); err == nil {
		s.Root = root // as pwd gives it
	}

	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	check(t, err)
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	check(t, err)
	clientPub, clientPriv, err := ed25519.GenerateKey(rand.Reader)
	check(t, err)
	clientSSHPub, err := ssh.NewPublicKey(clientPub)
	check(t, err)
	block, err := ssh.MarshalPrivateKey(clientPriv, "")
	check(t, err)
	check(t, os.WriteFile(filepath.Join(s.KeyPath, "id_rsa"), pem.EncodeToMemory(block), 0600))

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if string(key.Marshal()) == string(clientSSHPub.Marshal()) {
				return nil, nil
			}
			return nil, errors.New("sshtest: unknown key")
		},
	}
	config.AddHostKey(hostSigner)

	s.listener, err = net.Listen("tcp", "127.0.0.1:0")
	check(t, err)
	s.Addr = s.listener.Addr().String()
	line := knownhosts.Line([]string{knownhosts.Normalize(s.Addr)}, hostSigner.PublicKey())
	check(t, os.WriteFile(filepath.Join(s.KeyPath, "known_hosts"), []byte(line+"\n"), 0600))

	go s.serve(config)
	t.Cleanup(func() {
		s.listener.Close()
		s.closeConns()
	})
	return s
}

func check(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// serve accepts the connections to the server.
func (s *Server) serve(config *ssh.ServerConfig) {
	for {
		nc, err := s.listener.Accept()
		if err != nil {
			return
		}
		if s.down.Load() {
			nc.Close()
			continue
		}
		c := &conn{Conn: nc}
		s.connsMu.Lock()
		s.conns = append(s.conns, c)
		s.connsMu.Unlock()
		go func() {
			_, chans, reqs, err := ssh.NewServerConn(c, config)
			if err != nil {
				c.Close()
				return
			}
			go ssh.DiscardRequests(reqs)
			for newCh := range chans {
				if newCh.ChannelType() != "session" {
					newCh.Reject(ssh.UnknownChannelType, "only sessions are supported")
					continue
				}
				ch, reqs, err := newCh.Accept()
				if err != nil {
					continue
				}
				go s.session(ch, reqs)
			}
		}()
	}
}

// session serves the requests of a session: environment variables, a
// command run with sh, which can get signals, the shell, which is sh,
// and the SFTP subsystem.
func (s *Server) session(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	env := []string{"HOME=" + s.Root, "PATH=" + os.Getenv("PATH"), "SHELL=/bin/sh"}
	var cmd *exec.Cmd
	var tm *pty.Pty // the terminal of the command, if cols > 0
	cols, rows := 0, 0
	ended := make(chan struct{})
	defer func() {
		if cmd == nil {
			return
		}
		select {
		case <-ended:
		default: // the session was closed while the command runs: a hangup
			cmd.Process.Kill()
		}
	}()
	for {
		var req *ssh.Request
		select {
		case req = <-reqs:
		case <-ended: // the command ended
			return
		}
		if req == nil {
			return
		}
		switch req.Type {
		case "pty-req": // the client asks for a terminal for interactive commands
			var pr struct {
				Term       string
				Cols, Rows uint32
				W, H       uint32
				Modes      string
			}
			if s.Pty.Load() && ssh.Unmarshal(req.Payload, &pr) == nil {
				cols, rows = int(pr.Cols), int(pr.Rows)
			}
			req.Reply(true, nil)
		case "window-change":
			var wc struct{ Cols, Rows, W, H uint32 }
			if tm != nil && ssh.Unmarshal(req.Payload, &wc) == nil {
				tm.Resize(int(wc.Cols), int(wc.Rows))
			}
			req.Reply(true, nil)
		case "env":
			var kv struct{ Name, Value string }
			if ssh.Unmarshal(req.Payload, &kv) == nil {
				env = append(env, kv.Name+"="+kv.Value)
			}
			req.Reply(true, nil)
		case "signal":
			var sig struct{ Signal string }
			if ssh.Unmarshal(req.Payload, &sig) == nil && sig.Signal == string(ssh.SIGINT) && cmd != nil {
				cmd.Process.Signal(os.Interrupt)
			}
			req.Reply(true, nil)
		case "exec", "shell":
			var ex struct{ Command string }
			if req.Type == "shell" { // the shell of the user
				ex.Command = "exec $SHELL"
			} else if ssh.Unmarshal(req.Payload, &ex) != nil {
				req.Reply(false, nil)
				continue
			}
			if cmd != nil {
				req.Reply(false, nil)
				continue
			}
			cmd = exec.Command("sh", "-c", ex.Command)
			cmd.Dir, cmd.Env = s.Root, env
			wait := cmd.Wait
			if cols > 0 {
				var err error
				tm, err = pty.Start(cmd, cols, rows)
				if err != nil {
					req.Reply(false, nil)
					return
				}
				go io.Copy(tm, ch)
				output := make(chan struct{})
				go func() {
					io.Copy(ch, tm) // ends when the terminal is closed after the command
					close(output)
				}()
				wait = func() error {
					err := tm.Wait()
					// reading the output does not end on macOS when the
					// command ends, and it has to be read before closing
					time.AfterFunc(100*time.Millisecond, func() { tm.Close() })
					<-output
					return err
				}
			} else {
				cmd.Stdout, cmd.Stderr = ch, ch.Stderr()
				stdin, _ := cmd.StdinPipe()
				if err := cmd.Start(); err != nil {
					req.Reply(false, nil)
					return
				}
				go func() {
					io.Copy(stdin, ch)
					stdin.Close()
				}()
			}
			req.Reply(true, nil)
			go func() {
				code := 0
				if err := wait(); err != nil {
					code = 1
					var ee *exec.ExitError
					if errors.As(err, &ee) {
						code = ee.ExitCode()
					}
				}
				status := make([]byte, 4)
				binary.BigEndian.PutUint32(status, uint32(code))
				ch.SendRequest("exit-status", false, status)
				close(ended)
			}()
		case "subsystem":
			var sub struct{ Name string }
			if ssh.Unmarshal(req.Payload, &sub) != nil || sub.Name != "sftp" {
				req.Reply(false, nil)
				continue
			}
			req.Reply(true, nil)
			server, err := sftp.NewServer(ch, sftp.WithServerWorkingDirectory(s.Root))
			if err != nil {
				return
			}
			server.Serve()
			return
		default:
			req.Reply(false, nil)
		}
	}
}
