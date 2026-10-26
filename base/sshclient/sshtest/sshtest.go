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
	"testing"

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

	listener net.Listener
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
	t.Cleanup(func() { s.listener.Close() })
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
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go func() {
			_, chans, reqs, err := ssh.NewServerConn(conn, config)
			if err != nil {
				conn.Close()
				return
			}
			go ssh.DiscardRequests(reqs)
			for nc := range chans {
				if nc.ChannelType() != "session" {
					nc.Reject(ssh.UnknownChannelType, "only sessions are supported")
					continue
				}
				ch, reqs, err := nc.Accept()
				if err != nil {
					continue
				}
				go s.session(ch, reqs)
			}
		}()
	}
}

// session serves the requests of a session: environment variables, a
// command run with sh, and the SFTP subsystem.
func (s *Server) session(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	env := []string{"HOME=" + s.Root, "PATH=" + os.Getenv("PATH")}
	for req := range reqs {
		switch req.Type {
		case "pty-req", "window-change": // the client asks for a terminal for interactive commands
			req.Reply(true, nil)
		case "env":
			var kv struct{ Name, Value string }
			if ssh.Unmarshal(req.Payload, &kv) == nil {
				env = append(env, kv.Name+"="+kv.Value)
			}
			req.Reply(true, nil)
		case "exec":
			var cmd struct{ Command string }
			if ssh.Unmarshal(req.Payload, &cmd) != nil {
				req.Reply(false, nil)
				continue
			}
			req.Reply(true, nil)
			c := exec.Command("sh", "-c", cmd.Command)
			c.Dir, c.Env = s.Root, env
			c.Stdout, c.Stderr = ch, ch.Stderr()
			stdin, _ := c.StdinPipe()
			go func() {
				io.Copy(stdin, ch)
				stdin.Close()
			}()
			code := 0
			if err := c.Run(); err != nil {
				code = 1
				var ee *exec.ExitError
				if errors.As(err, &ee) {
					code = ee.ExitCode()
				}
			}
			status := make([]byte, 4)
			binary.BigEndian.PutUint32(status, uint32(code))
			ch.SendRequest("exit-status", false, status)
			return
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
