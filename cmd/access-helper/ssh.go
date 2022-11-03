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
	UserNameKey = "netcon.janog.gr.jp/userName"
	PasswordKey = "netcon.janog.gr.jp/password"
	PortKey     = "netcon.janog.gr.jp/port"
)

type SSHAccessHelper struct {
}

func (h *SSHAccessHelper) loadParameters(
	nodeDefinition containerlab.NodeDefinition,
) (string, string, uint16, error) {
	userName, ok := nodeDefinition.Labels[UserNameKey]
	if !ok {
		return "", "", 0, errors.New("username not found")
	}

	password, ok := nodeDefinition.Labels[PasswordKey]
	if !ok {
		return "", "", 0, errors.New("password not found")
	}

	portRaw, ok := nodeDefinition.Labels[PortKey]
	if !ok {
		return "", "", 0, errors.New("port not found")
	}

	port, err := strconv.ParseUint(portRaw, 10, 16)
	return userName, password, uint16(port), err
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

	config := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
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
