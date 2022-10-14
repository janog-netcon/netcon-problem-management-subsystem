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

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"

	"github.com/pkg/errors"
	gossh "golang.org/x/crypto/ssh"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

type SSHServer struct {
	client.Client
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

func (r *SSHServer) handlePasswordAuthentication(ctx context.Context, sCtx ssh.Context, password string) (bool, error) {
	return true, nil
}

func (r *SSHServer) handle(ctx context.Context, s ssh.Session) {
	cmd := exec.Command("access-helper", "-t", "/data/pro-001-a0e832/manifest.yml")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return
	}

	go func() { _, _ = io.Copy(ptmx, s) }()
	_, _ = io.Copy(s, ptmx)

	s.Close()
}

func (r *SSHServer) Start(ctx context.Context) error {
	log := log.FromContext(ctx)

	server := &ssh.Server{
		Addr: ":2222",
		PasswordHandler: func(sctx ssh.Context, password string) bool {
			ok, err := r.handlePasswordAuthentication(ctx, sctx, password)
			if err != nil {
				log.Error(err, "failed to handle password-based authentication")
				return false
			}
			return ok
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
