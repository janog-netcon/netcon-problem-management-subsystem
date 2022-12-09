package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/containerlab"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func init() {
	registerAccessHelper(AccessMethodSSH, &SSHAccessHelper{})
}

const AccessMethodSSH AccessMethod = "ssh"

const (
	usernameKey = "netcon.janog.gr.jp/username"

	passwordKey = "netcon.janog.gr.jp/password"

	portKey            = "netcon.janog.gr.jp/port"
	defaultPort uint16 = 22
)

// defaultUsername and defaultPassword should be override when building access-helper with -ldflags
// ref: https://uokada.hatenablog.jp/entry/2015/05/20/001208
var (
	defaultUsername string
	defaultPassword string
)

type SSHAccessHelper struct {
}

func (h *SSHAccessHelper) loadParameters(
	nodeDefinition containerlab.NodeDefinition,
) (string, string, uint16, error) {
	username := defaultUsername
	if v, ok := nodeDefinition.Labels[usernameKey]; ok {
		username = v
	}

	password := defaultPassword
	if v, ok := nodeDefinition.Labels[passwordKey]; ok {
		password = v
	}

	port := defaultPort
	if v, ok := nodeDefinition.Labels[portKey]; ok {
		p, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return "", "", 0, errors.Errorf("could not parse port")
		}
		port = uint16(p)
	}

	return username, password, port, nil
}

func (h *SSHAccessHelper) _access(
	ctx context.Context,
	nodeDefinition containerlab.NodeDefinition,
	containerDetails containerlab.ContainerDetails,
) (int, error) {
	userName, password, port, err := h.loadParameters(nodeDefinition)
	if err != nil {
		return 0, err
	}

	// TODO: change ClientConfig for node kind
	config := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i, _ := range answers {
					answers[i] = password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	serverAddress := containerDetails.IPv4Address
	serverAddress = serverAddress[:strings.Index(serverAddress, "/")]
	serverAddress = fmt.Sprintf("%s:%d", serverAddress, port)

	client, err := ssh.Dial("tcp", serverAddress, config)
	if err != nil {
		return 0, errors.Wrap(err, "failed to access node via SSH")
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return 0, errors.Wrap(err, "failed to open new session")
	}
	defer session.Close()

	width, height, err := term.GetSize(0)
	if err != nil {
		return 0, err
	}

	state, err := term.MakeRaw(0)
	if err != nil {
		return 0, err
	}
	defer term.Restore(0, state)

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO: 1,
	}

	// Request PTY for this session
	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		return 0, err
	}

	// Window size synchronization
	sigWinchChan := make(chan os.Signal, 1)
	signal.Notify(sigWinchChan, syscall.SIGWINCH)
	go func() {
		for range sigWinchChan {
			width, height, err := term.GetSize(0)
			if err != nil {
				continue
			}

			session.WindowChange(height, width)
		}
	}()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	if err := session.Shell(); err != nil {
		return 0, err
	}

	err = session.Wait()
	if err, ok := err.(*ssh.ExitError); ok {
		return err.ExitStatus(), nil
	}

	return 0, err
}

func (h *SSHAccessHelper) access(
	ctx context.Context,
	nodeDefinition containerlab.NodeDefinition,
	containerDetails containerlab.ContainerDetails,
) error {
	status, err := h._access(ctx, nodeDefinition, containerDetails)
	if err != nil {
		return err
	}

	// os.Exit will not return, so return statement will not be executed
	os.Exit(status)
	return nil
}
