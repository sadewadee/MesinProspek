package main

import (
	"bytes"
	"context"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want int
	}{
		{name: "no command", args: nil, want: 2},
		{name: "unknown command", args: []string{"nope"}, want: 2},
		{name: "not implemented yet", args: []string{"worker"}, want: 2},
		{name: "invalid config", args: []string{"api"}, env: map[string]string{"MSP_ENV": "prod"}, want: 1},
		{name: "api stops on cancel", args: []string{"api"}, env: map[string]string{"MSP_HTTP_ADDR": "127.0.0.1:0"}, want: 0},
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
