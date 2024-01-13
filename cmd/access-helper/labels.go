package main

// These are the label keys that we can customize how to access Node in ContainerLab.
const (
	// AccessMethodKey is the label key to specify the access method.
	// The possible values are either "ssh" or "exec".
	// With "ssh", access-helper will try to connect to the Node via SSH.
	// With "exec", access-helper will try to connect to the Node with docker exec.
	AccessMethodKey = "netcon.janog.gr.jp/accessMethod"

	// AdminOnlyKey is the label key to specify the Node visibility for participants.
	// If the value is "true", the Node is not visible for participants.
	AdminOnlyKey = "netcon.janog.gr.jp/adminOnly"

	// sshUsernameKey is the label key to specify username for SSH access.
	// The default value is defaultUsername.
	sshUsernameKey = "netcon.janog.gr.jp/sshUsername"

	// sshPasswordKey is the label key to specify password for SSH access.
	// The default value is defaultPassword.
	sshPasswordKey = "netcon.janog.gr.jp/sshPassword"

	// sshUsernameForAdminKey is the label key to specify username for SSH access for admin user.
	// The default value is defaultUsername.
	sshUsernameForAdminKey = "netcon.janog.gr.jp/sshUsernameForAdmin"

	// sshPasswordForAdminKey is the label key to specify password for SSH access for admin user.
	// The default value is defaultPassword.
	sshPasswordForAdminKey = "netcon.janog.gr.jp/sshPasswordForAdmin"

	// sshPortKey is the label key to specify port number for SSH access.
	// The default value is defaultPort.
	sshPortKey            = "netcon.janog.gr.jp/sshPort"
	defaultSSHPort uint16 = 22

	// execCommandKey is the label key to specify command for exec access.
	// The default value is defaultExecCommand.
	execCommandKey     = "netcon.janog.gr.jp/execCommand"
	defaultExecCommand = "sh"
)

var (
	// defaultUsername is a default username for SSH access.
	// You can override this value when building access-helper with -ldflags.
	// ref: https://uokada.hatenablog.jp/entry/2015/05/20/001208
	defaultSSHUsername string

	// defaultPassword is a default password for SSH access.
	// You can override this value when building access-helper with -ldflags.
	// ref: https://uokada.hatenablog.jp/entry/2015/05/20/001208
	defaultSSHPassword string
)
