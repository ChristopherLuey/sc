package ssh

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	conn *ssh.Client
	mu   sync.Mutex
	done chan struct{}
}

// Dial connects to the SSH server.
func Dial(host string, port int, user string, useAgent bool, identityFile string) (*Client, error) {
	host, user, identityFile, port = ResolveHost(host, user, identityFile, port)

	methods := BuildAuthMethods(useAgent, identityFile)
	if len(methods) == 0 {
		return nil, fmt.Errorf("no SSH auth methods available")
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            methods,
		HostKeyCallback: HostKeyCallback(),
		Timeout:         15 * time.Second,
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}

	c := &Client{conn: conn, done: make(chan struct{})}
	go c.keepalive()
	return c, nil
}

// RunCommand executes a command on the remote host.
func (c *Client) RunCommand(cmd string) (string, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return "", "", fmt.Errorf("not connected")
	}

	sess, err := c.conn.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	var stdout, stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = &stderr

	err = sess.Run(cmd)
	return stdout.String(), stderr.String(), err
}

func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	close(c.done)
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *Client) keepalive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.Lock()
			if c.conn != nil {
				_, _, _ = c.conn.SendRequest("keepalive@openssh.com", true, nil)
			}
			c.mu.Unlock()
		}
	}
}
