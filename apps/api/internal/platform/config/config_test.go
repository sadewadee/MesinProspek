package config

import (
	"log/slog"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			want: Config{Env: "local", LogLevel: slog.LevelInfo, HTTPAddr: ":8080"},
		},
		{
			name: "explicit values",
			env:  map[string]string{"MSP_ENV": "production", "MSP_LOG_LEVEL": "debug", "MSP_HTTP_ADDR": ":9000"},
			want: Config{Env: "production", LogLevel: slog.LevelDebug, HTTPAddr: ":9000"},
		},
		{
			name:    "unknown env",
			env:     map[string]string{"MSP_ENV": "prod"},
			wantErr: true,
		},
		{
			name:    "unknown log level",
			env:     map[string]string{"MSP_LOG_LEVEL": "loud"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse(func(k string) string { return tt.env[k] })
			if (err != nil) != tt.wantErr {
				t.Fatalf("parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
