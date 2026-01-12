package tracing

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"go.opentelemetry.io/otel/propagation"
)

const (
	envCarrierPrefix = "OTEL_ENV_CARRIER_"
)

type EnvCarrier map[string]string

var _ propagation.TextMapCarrier = EnvCarrier{}

// Get implements [propagation.TextMapCarrier].
func (EnvCarrier) Get(key string) string {
	return os.Getenv(envCarrierPrefix + key)
}

// Keys implements [propagation.TextMapCarrier].
func (EnvCarrier) Keys() []string {
	keys := []string{}
	for _, key := range os.Environ() {
		if strings.HasPrefix(key, envCarrierPrefix) {
			keys = append(keys, key[len(envCarrierPrefix):])
		}
	}
	return keys
}

// Set implements [propagation.TextMapCarrier].
func (c EnvCarrier) Set(key string, value string) {
	c[key] = value
}

func (c EnvCarrier) InjectToCmd(cmd *exec.Cmd) {
	for key, value := range c {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s%s=%s", envCarrierPrefix, key, value))
	}
}
