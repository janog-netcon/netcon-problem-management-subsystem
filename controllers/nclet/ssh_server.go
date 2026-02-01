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
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pkg/errors"
	gossh "golang.org/x/crypto/ssh"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/internal/ssh"
	"github.com/janog-netcon/netcon-problem-management-subsystem/internal/tracing"
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

	adminPassword string
}

func NewSSHServer(client client.Client, sshAddr string, adminPassword string) *SSHServer {
	return &SSHServer{
		Client:        client,
		sshAddr:       sshAddr,
		adminPassword: adminPassword,
	}
}

var _ manager.Runnable = &SSHServer{}

var rsaHostKeyPath = path.Join("data", "ssh_host_rsa_key")

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

func (r *SSHServer) handlePasswordAuthentication(ctx context.Context, sCtx ssh.Context, password string) error {
	span := tracing.SpanFromContext(ctx)

	user, err := parseUser(sCtx.User())
	if err != nil {
		return errors.New("invalid user format")
	}

	if user.Admin {
		span.AddEvent("SSH request from admin received")
		if password != r.adminPassword {
			return errors.New("invalid password")
		}
		return nil
	}

	span.AddEvent("SSH request from participants received")

	problemEnvironment := netconv1alpha1.ProblemEnvironment{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: "netcon",
		Name:      user.ProblemEnvironmentName,
	}, &problemEnvironment); err != nil {
		return errors.New("problem environment not found")
	}

	if util.GetProblemEnvironmentCondition(
		&problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionAssigned,
	) != metav1.ConditionTrue {
		return errors.New("problem environment not assigned")
	}

	if password != problemEnvironment.Status.Password {
		return errors.New("invalid password")
	}

	return nil
}

func (r *SSHServer) handle(ctx context.Context, s ssh.Session) error {
	span := tracing.SpanFromContext(ctx)

	if s.RawCommand() != "" {
		return tracing.GenerateError(span, "command execution is not supported")
	}

	otlpEndpoint := os.Getenv(tracing.OTLPEndpointKey)

	user, err := parseUser(s.User())
	if err != nil {
		return tracing.GenerateError(span, "invalid user format")
	}

	topologyFilePath := path.Join("data", user.ProblemEnvironmentName, "manifest.yml")

	args := []string{"-t", topologyFilePath}

	if user.Admin {
		args = append(args, "--admin")
	}
	if otlpEndpoint != "" {
		args = append(args, "--enable-otel")
	}

	if user.NodeName != "" {
		args = append(args, user.NodeName)
	}

	cmd := exec.CommandContext(ctx, "access-helper", args...)

	if otlpEndpoint != "" {
		cmd.Env = append(syscall.Environ(), fmt.Sprintf("%s=%s", tracing.OTLPEndpointKey, otlpEndpoint))

		carrier := tracing.EnvCarrier{}
		otel.GetTextMapPropagator().Inject(ctx, carrier)
		carrier.InjectToCmd(cmd)
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return tracing.WrapError(span, err, "failed to start pty")
	}
	defer ptmx.Close()

	// TODO: handle terminal resize

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(ptmx, s)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(s, ptmx)
	}()

	// case 1.) Command finished
	//   In this case, cmd.Wait() will return first. Then, we will close the session.
	// case 2.) Context done (Idle timer expired, underlying stream closed)
	//   In this case, ctx.Done() will return first. Then, we will close the command.
	if err := cmd.Wait(); err != nil {
		span.SetStatus(codes.Error, "failed to wait for session")
		span.RecordError(err)
	}

	// Close the session to signal the client that the session is finished.
	if err := s.Close(); err != nil {
		span.SetStatus(codes.Error, "failed to close session")
		span.RecordError(err)
	}
	wg.Wait()

	return nil
}

func (r *SSHServer) Start(ctx context.Context) error {
	_ = log.FromContext(ctx)

	server := &ssh.Server{
		Addr:        r.sshAddr,
		IdleTimeout: 20 * time.Minute,
		PasswordHandler: func(sctx ssh.Context, password string) bool {
			ctx, span := tracing.Tracer.Start(sctx, "Server#PasswordHandler")
			defer span.End()

			remoteAddr := strings.Split(sctx.RemoteAddr().String(), ":")[0]
			log := log.FromContext(ctx)

			if err := r.handlePasswordAuthentication(ctx, sctx, password); err != nil {
				msg := "SSH authentication failed"
				log.Info(msg, "remoteAddr", remoteAddr, "user", sctx.User(), "reason", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, msg)
				sshAuthTotalFailed.Inc()
				return false
			}

			msg := "SSH authentication succeeded"
			log.Info(msg, "remoteAddr", remoteAddr, "user", sctx.User())
			span.AddEvent(msg)
			sshAuthTotalSucceeded.Inc()
			return true
		},
		Handler: func(s ssh.Session) {
			defer s.Close()

			sshSessionsInFlight.Inc()
			defer sshSessionsInFlight.Dec()

			ctx, span := tracing.Tracer.Start(s.Context(), "Server#Handler")
			defer span.End()

			remoteAddr := strings.Split(s.RemoteAddr().String(), ":")[0]
			log := log.FromContext(ctx)

			start := time.Now()

			if err := r.handle(ctx, s); err != nil {
				fmt.Fprintf(s, "Internal error occured. Please contact NETCON members with Session ID and Trace ID.\n")

				msg := "SSH session finished with error"
				duration := time.Since(start).Seconds()
				log.Info(msg, "remoteAddr", remoteAddr, "user", s.User(), "durationSecond", duration, "reason", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, msg)
				sshSessionDuration.Observe(duration)
				return
			}

			msg := "SSH session finished successfully"
			duration := time.Since(start).Seconds()
			log.Info(msg, "remoteAddr", remoteAddr, "user", s.User(), "durationSecond", duration)
			span.AddEvent(msg)
			sshSessionDuration.Observe(duration)
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
