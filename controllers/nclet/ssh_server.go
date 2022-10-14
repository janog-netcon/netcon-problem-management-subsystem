package controllers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
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

func (r *SSHServer) ensureRSAHostKey(ctx context.Context) error {
	log := log.FromContext(ctx)

	if r.fileExists(rsaHostKeyPath) {
		log.V(1).Info("RSA host key already exists, skip to create RSA host key")
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

func (r *SSHServer) ensureHostKeys(ctx context.Context) error {
	if err := r.ensureRSAHostKey(ctx); err != nil {
		return fmt.Errorf("failed to ensure RSA host key: %w", err)
	}

	return nil
}

func (r *SSHServer) injectHostKeys(ctx context.Context, config *ssh.ServerConfig) error {
	if err := r.ensureHostKeys(ctx); err != nil {
		return fmt.Errorf("failed to ensure host keys: %w", err)
	}

	rsaHostKeyData, err := os.ReadFile(rsaHostKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read RSA host key: %w", err)
	}

	rsaHostKey, err := ssh.ParsePrivateKey(rsaHostKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse RSA host key: %w", err)
	}

	config.AddHostKey(rsaHostKey)

	return nil
}

func (r *SSHServer) bannerCallback(ctx context.Context, conn ssh.ConnMetadata) string {
	return "network contest"
}

func (r *SSHServer) passwordCallback(ctx context.Context, conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	return nil, nil
}

func (r *SSHServer) handleChannel(ctx context.Context, newChannel ssh.NewChannel) {
	if newChannel.ChannelType() != "session" {
		newChannel.Reject(ssh.UnknownChannelType, "session channel is only accepted")
		return
	}

	channel, reqs, err := newChannel.Accept()
	if err != nil {
		return
	}
	go ssh.DiscardRequests(reqs)

	// TODO: access to problem environment
	channel.Close()
}

func (r *SSHServer) handleChannels(ctx context.Context, chans <-chan ssh.NewChannel) {
	for channel := range chans {
		r.handleChannel(ctx, channel)
	}
}

func (r *SSHServer) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", "0.0.0.0:22222")
	if err != nil {
		return err
	}

	config := &ssh.ServerConfig{
		BannerCallback: func(conn ssh.ConnMetadata) string {
			return r.bannerCallback(ctx, conn)
		},
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			return r.passwordCallback(ctx, conn, password)
		},
	}

	if err := r.injectHostKeys(ctx, config); err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		_, chans, reqs, err := ssh.NewServerConn(conn, config)
		if err != nil {
			return err
		}

		go ssh.DiscardRequests(reqs)
		go r.handleChannels(ctx, chans)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *SSHServer) SetupWithManager(mgr ctrl.Manager) error {
	return nil
}
