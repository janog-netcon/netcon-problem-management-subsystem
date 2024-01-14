package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
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

type SSHAccessHelper struct{}

func (h *SSHAccessHelper) loadParameters(
	nodeDefinition containerlab.NodeDefinition,
	isAdmin bool,
) (string, string, uint16, error) {
	username := defaultSSHUsername
	if v, ok := nodeDefinition.Labels[sshUsernameForAdminKey]; ok && isAdmin {
		username = v
	} else if v, ok := nodeDefinition.Labels[sshUsernameKey]; ok {
		username = v
	}

	password := defaultSSHPassword
	if v, ok := nodeDefinition.Labels[sshPasswordForAdminKey]; ok && isAdmin {
		password = v
	} else if v, ok := nodeDefinition.Labels[sshPasswordKey]; ok {
		password = v
	}

	port := defaultSSHPort
	if v, ok := nodeDefinition.Labels[sshPortKey]; ok {
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
	isAdmin bool,
) (int, error) {
	userName, password, port, err := h.loadParameters(nodeDefinition, isAdmin)
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
				for i := range answers {
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

	state, err := makeRaw(0)
	if err != nil {
		return 0, err
	}
	defer restore(0, state)

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:    1,
		ssh.ECHOCTL: 1,
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
	defer func() {
		signal.Stop(sigWinchChan)
		close(sigWinchChan)
	}()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// If we copy stdin to the session with `session.Stdin = os.Stdin`, ssh package will copy bytes with io.Copy.
	// However, it occurs a problem that os.Stdin can't be used after the session is closed until the enter is pressed.
	//
	// 1. The user closes SSH session between access-helper and the node.
	//    But io.Copy is not stopped because any characters are not sent to access-helper.
	// 2. term.Restore() changes terminal mode from non-canonical mode to canonical mode.
	// 3. Until the user presses enter, the entered characters are buffered and not sent to access-helper.
	// 4. If the user presses enter, the entered characters are delivered to io.Copy.
	//    However, because the SSH session is already closed, the characters are discarded.
	// 5. The user becomes able to select next nodes normally.
	//
	// To avoid this problem, this function will use special I/O handler for stdin.

	// To prevent ssh package to use io.Copy, call StdinPipe().
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return 0, err
	}

	isConnectionClosed := false
	stdinClonerDone := sync.WaitGroup{}
	stdinClonerDone.Add(1)

	defer func() {
		isConnectionClosed = true
		stdinClonerDone.Wait()
	}()

	go func() {
		buf := make([]byte, 32*1024)
		for {
			if isConnectionClosed {
				break
			}

			nr, err := os.Stdin.Read(buf)
			if err != nil && !errors.Is(err, io.EOF) {
				fmt.Printf("Read: %+v\n", err)
				break
			}

			if nr == 0 {
				continue
			}

			if _, err := stdinPipe.Write(buf[:nr]); err != nil {
				fmt.Printf("Write: %+v\n", err)
				break
			}
		}

		stdinClonerDone.Done()
	}()

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
	isAdmin bool,
) error {
	_, err := h._access(ctx, nodeDefinition, containerDetails, isAdmin)
	if err != nil {
		return err
	}

	return nil
}
