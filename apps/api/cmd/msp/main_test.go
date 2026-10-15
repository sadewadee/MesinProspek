package main

import (
	"bytes"
	"context"
	"testing"
)

// apiEnv is a valid environment of `msp api` with key set to value. Nothing
// listens on the database and Kratos addresses: `msp api` starts without them.
func apiEnv(key, value string) map[string]string {
	env := map[string]string{
		"MSP_HTTP_ADDR":         "127.0.0.1:0",
		"MSP_DATABASE_URL":      "postgres://msp_app@127.0.0.1:1/msp",
		"MSP_PUBLIC_URL":        "http://localhost:8000",
		"MSP_KRATOS_PUBLIC_URL": "http://127.0.0.1:1",
	}
	env[key] = value
	return env
}

func TestRun(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want int
	}{
		{name: "no command", args: nil, want: 2},
		{name: "unknown command", args: []string{"nope"}, want: 2},
		{name: "not implemented yet", args: []string{"tool"}, want: 2},
		{name: "worker stops on cancel", args: []string{"worker"}, want: 0},
		{name: "invalid config", args: []string{"api"}, env: map[string]string{"MSP_ENV": "prod"}, want: 1},
		{name: "api without database url", args: []string{"api"}, env: apiEnv("MSP_DATABASE_URL", ""), want: 1},
		{name: "api with invalid database url", args: []string{"api"}, env: apiEnv("MSP_DATABASE_URL", "postgres://%zz"), want: 1},
		{name: "api without kratos url", args: []string{"api"}, env: apiEnv("MSP_KRATOS_PUBLIC_URL", ""), want: 1},
		{name: "api stops on cancel", args: []string{"api"}, env: apiEnv("MSP_HTTP_ADDR", "127.0.0.1:0"), want: 0},
		{name: "migrate without database url", args: []string{"migrate"}, env: map[string]string{"MSP_MIGRATE_DATABASE_URL": ""}, want: 1},
		{name: "migrate with invalid database url", args: []string{"migrate"}, env: map[string]string{"MSP_MIGRATE_DATABASE_URL": "postgres://%zz"}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // api returns as soon as it has started
			var stdout, stderr bytes.Buffer
			if got := run(ctx, tt.args, &stdout, &stderr); got != tt.want {
				t.Fatalf("run() = %d, want %d; stderr=%q stdout=%q", got, tt.want, stderr.String(), stdout.String())
			}
		})
	}
}
