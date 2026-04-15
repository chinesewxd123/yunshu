package sshclient

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type AuthType string

const (
	AuthPassword AuthType = "password"
	AuthKey      AuthType = "key"
)

type Config struct {
	Host     string
	Port     int
	Username string
	AuthType AuthType

	Password   string
	PrivateKey string
	Passphrase string

	ConnectTimeout time.Duration
	CommandTimeout time.Duration
}

type Client struct {
	cfg    Config
	client *ssh.Client
}

func Dial(ctx context.Context, cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.Host) == "" || strings.TrimSpace(cfg.Username) == "" {
		return nil, errors.New("host and username are required")
	}
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}
	if cfg.CommandTimeout <= 0 {
		cfg.CommandTimeout = 15 * time.Second
	}

	authMethods, err := buildAuthMethods(cfg)
	if err != nil {
		return nil, err
	}
	sshCfg := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // MVP: allow; should be replaced by known_hosts verification
		Timeout:         cfg.ConnectTimeout,
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	dialer := net.Dialer{Timeout: cfg.ConnectTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	cc, chans, reqs, err := ssh.NewClientConn(conn, addr, sshCfg)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Client{cfg: cfg, client: ssh.NewClient(cc, chans, reqs)}, nil
}

func buildAuthMethods(cfg Config) ([]ssh.AuthMethod, error) {
	switch cfg.AuthType {
	case "", AuthPassword:
		if strings.TrimSpace(cfg.Password) == "" {
			return nil, errors.New("password is required")
		}
		// Some SSH servers disable direct "password" but enable keyboard-interactive.
		// Provide both to improve compatibility.
		return []ssh.AuthMethod{
			ssh.Password(cfg.Password),
			ssh.KeyboardInteractive(func(_ string, _ string, questions []string, _ []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = cfg.Password
				}
				return answers, nil
			}),
		}, nil
	case AuthKey:
		if strings.TrimSpace(cfg.PrivateKey) == "" {
			return nil, errors.New("private key is required")
		}
		var signer ssh.Signer
		var err error
		if strings.TrimSpace(cfg.Passphrase) != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(cfg.PrivateKey), []byte(cfg.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
		}
		if err != nil {
			return nil, err
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	default:
		return nil, errors.New("unknown auth_type")
	}
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

type ExecResult struct {
	Stdout    string
	Stderr    string
	ExitCode  int
	Duration  time.Duration
	Truncated bool
}

// Exec runs a command and returns stdout/stderr (with max bytes to avoid memory blow).
func (c *Client) Exec(ctx context.Context, cmd string, maxBytes int) (ExecResult, error) {
	if c == nil || c.client == nil {
		return ExecResult{}, errors.New("ssh client not connected")
	}
	if strings.TrimSpace(cmd) == "" {
		return ExecResult{}, errors.New("cmd is empty")
	}
	if maxBytes <= 0 {
		maxBytes = 64 * 1024
	}

	start := time.Now()
	session, err := c.client.NewSession()
	if err != nil {
		return ExecResult{}, err
	}
	defer session.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutW := &limitedWriter{W: &stdoutBuf, N: maxBytes}
	stderrW := &limitedWriter{W: &stderrBuf, N: maxBytes}
	session.Stdout = stdoutW
	session.Stderr = stderrW

	done := make(chan error, 1)
	go func() { done <- session.Run(cmd) }()

	var runErr error
	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		runErr = ctx.Err()
	case err := <-done:
		runErr = err
	}

	exitCode := 0
	if runErr != nil {
		var ee *ssh.ExitError
		if errors.As(runErr, &ee) {
			exitCode = ee.ExitStatus()
			// treat as non-fatal for result, caller can decide
		} else {
			return ExecResult{}, runErr
		}
	}
	return ExecResult{
		Stdout:    stdoutBuf.String(),
		Stderr:    stderrBuf.String(),
		ExitCode:  exitCode,
		Duration:  time.Since(start),
		Truncated: stdoutW.Truncated || stderrW.Truncated,
	}, nil
}

// StreamLines runs a long-lived command and streams lines to handler.
// Caller should cancel ctx to stop.
func (c *Client) StreamLines(ctx context.Context, cmd string, onLine func(line string)) error {
	if c == nil || c.client == nil {
		return errors.New("ssh client not connected")
	}
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		_ = session.Close()
		return err
	}

	if err := session.Start(cmd); err != nil {
		_ = session.Close()
		return err
	}

	errCh := make(chan error, 2)
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			onLine(sc.Text())
		}
		errCh <- sc.Err()
	}()
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			onLine(sc.Text())
		}
		errCh <- sc.Err()
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		_ = session.Close()
		return ctx.Err()
	case e := <-errCh:
		_ = session.Close()
		if e == io.EOF || e == nil {
			return nil
		}
		return e
	}
}

type limitedWriter struct {
	W         io.Writer
	N         int
	Truncated bool
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.N <= 0 {
		w.Truncated = true
		return len(p), nil
	}
	if len(p) > w.N {
		w.Truncated = true
		p = p[:w.N]
	}
	n, err := w.W.Write(p)
	w.N -= n
	return len(p), err
}
