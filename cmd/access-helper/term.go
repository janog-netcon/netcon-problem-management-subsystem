package main

import "golang.org/x/sys/unix"

type State struct {
	termios unix.Termios
}

func makeRaw(fd int) (*State, error) {
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}

	oldState := State{termios: *termios}

	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8
	termios.Cc[unix.VMIN] = 0
	termios.Cc[unix.VTIME] = 1
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, termios); err != nil {
		return nil, err
	}

	return &oldState, nil
}

func restore(fd int, state *State) error {
	return unix.IoctlSetTermios(fd, unix.TCSETS, &state.termios)
}
