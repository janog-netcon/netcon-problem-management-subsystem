package controllers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pkg/errors"
	gossh "golang.org/x/crypto/ssh"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
)

type User struct {
	ProblemEnvironmentName string
	NodeName               string
	Admin                  bool
}

// parseUser parses user name and return the information of connection
//
// valid formats are the following:
// * Access with prompt
//   - nc_{{ PROBLEM_ENVIRONMENT_NAME }}
//   - ncadmin_{{ PROBLEM_ENVIRONMENT_NAME }}
//
// * Access without prompt
//   - nc_{{ PROBLEM_ENVIRONMENT_NAME }}_{{ NODE_NAME }}
//   - ncadmin_{{ PROBLEM_ENVIRONMENT_NAME }}_{{ NODE_NAME }}
func parseUser(user string) (*User, error) {
	parts := strings.Split(user, "_")

	if !(len(parts) == 2 || len(parts) == 3) {
		return nil, errors.New("invalid format")
	}

	if !(parts[0] == "nc" || parts[0] == "ncadmin") {
		return nil, errors.New("invalid format")
	}

	problemEnvironmentName := parts[1]
	nodeName := ""
	if len(parts) == 3 {
		nodeName = parts[2]
	}
	admin := parts[0] == "ncadmin"

	return &User{
		ProblemEnvironmentName: problemEnvironmentName,
		NodeName:               nodeName,
		Admin:                  admin,
	}, nil
}

type SSHServer struct {
	client.Client

	sshAddr string
}

func NewSSHServer(sshAddr string) *SSHServer {
	return &SSHServer{
		sshAddr: sshAddr,
	}

}

var _ manager.Runnable = &SSHServer{}
var _ inject.Client = &SSHServer{}

var rsaHostKeyPath = path.Join("data", "ssh_host_rsa_key")

func (r *SSHServer) InjectClient(client client.Client) error {
	r.Client = client
	return nil
}

func (r *SSHServer) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *SSHServer) ensureRSAHostKey() error {
	if r.fileExists(rsaHostKeyPath) {
		return nil
	}

	rsaHostKeyFile, err := os.Create(rsaHostKeyPath)
	if err != nil {
		return errors.Wrap(err, "failed to create file for RSA host key")
	}

	rsaHostKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return errors.Wrap(err, "failed to generate RSA host key")
	}

	if err := pem.Encode(rsaHostKeyFile, &pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(rsaHostKey),
	}); err != nil {
		return errors.Wrap(err, "failed to encode RSA host key")
	}

	return nil

}

func (r *SSHServer) ensureHostKeys() error {
	if err := r.ensureRSAHostKey(); err != nil {
		return fmt.Errorf("failed to ensure RSA host key: %w", err)
	}

	return nil
}

func (r *SSHServer) injectHostKeys(server *ssh.Server) error {
	if err := r.ensureHostKeys(); err != nil {
		return fmt.Errorf("failed to ensure host keys: %w", err)
	}

	rsaHostKeyData, err := os.ReadFile(rsaHostKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read RSA host key: %w", err)
	}

	rsaHostKey, err := gossh.ParsePrivateKey(rsaHostKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse RSA host key: %w", err)
	}

	server.AddHostKey(rsaHostKey)

	return nil
}

func (r *SSHServer) handlePasswordAuthentication(ctx context.Context, sCtx ssh.Context, password string) bool {
	user, err := parseUser(sCtx.User())
	if err != nil {
		return false
	}

	if user.Admin {
		// TODO: password authentication for user
		return true
	} else {
		problemEnvironment := netconv1alpha1.ProblemEnvironment{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: "netcon",
			Name:      user.ProblemEnvironmentName,
		}, &problemEnvironment); err != nil {
			return false
		}

		if util.GetProblemEnvironmentCondition(
			&problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionAssigned,
		) != metav1.ConditionTrue {
			// TODO(proelbtn): Gateway is not found now. So, I bypassed "Assigned" check
		}

		if problemEnvironment.Status.Password != password {
			return false
		}

		return true
	}
}

func (r *SSHServer) handle(ctx context.Context, s ssh.Session) {
	user, err := parseUser(s.User())
	if err != nil {
		return
	}

	topologyFilePath := path.Join("data", user.ProblemEnvironmentName, "manifest.yml")

	args := []string{"-t", topologyFilePath}

	if user.Admin {
		args = append(args, "--admin")
	}

	if user.NodeName != "" {
		args = append(args, user.NodeName)
	}

	cmd := exec.Command("access-helper", args...)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return
	}

	// TODO: handle terminal resize

	go func() { _, _ = io.Copy(ptmx, s) }()
	_, _ = io.Copy(s, ptmx)

	s.Close()
}

func (r *SSHServer) Start(ctx context.Context) error {
	_ = log.FromContext(ctx)

	server := &ssh.Server{
		Addr: r.sshAddr,
		PasswordHandler: func(sctx ssh.Context, password string) bool {
			// TODO: Add logging
			return r.handlePasswordAuthentication(ctx, sctx, password)
		},
		Handler: func(s ssh.Session) {
			r.handle(ctx, s)
		},
	}

	if err := r.injectHostKeys(server); err != nil {
		return err
	}

	if err := server.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SSHServer) SetupWithManager(mgr ctrl.Manager) error {
	return nil
}
