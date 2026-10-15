package config

import (
	"log/slog"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		command string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{
			name:    "defaults",
			command: "api",
			env:     map[string]string{},
			want:    Config{Env: "local", LogLevel: slog.LevelInfo, HTTPAddr: ":8080"},
		},
		{
			name:    "explicit values",
			command: "api",
			env: map[string]string{
				"MSP_ENV": "production", "MSP_LOG_LEVEL": "debug", "MSP_HTTP_ADDR": ":9000",
				"MSP_DATABASE_URL":         "postgres://msp_app@db/msp",
				"MSP_MIGRATE_DATABASE_URL": " postgres://msp_owner@db/msp ",
			},
			want: Config{
				Env: "production", LogLevel: slog.LevelDebug, HTTPAddr: ":9000",
				DatabaseURL:        "postgres://msp_app@db/msp",
				MigrateDatabaseURL: "postgres://msp_owner@db/msp",
			},
		},
		{
			name:    "unknown env",
			command: "api",
			env:     map[string]string{"MSP_ENV": "prod"},
			wantErr: true,
		},
		{
			name:    "unknown log level",
			command: "api",
			env:     map[string]string{"MSP_LOG_LEVEL": "loud"},
			wantErr: true,
		},
		{
			name:    "api does not need database urls yet",
			command: "api",
			env:     map[string]string{"MSP_MIGRATE_DATABASE_URL": ""},
			want:    Config{Env: "local", LogLevel: slog.LevelInfo, HTTPAddr: ":8080"},
		},
		{
			name:    "migrate needs the migrate url",
			command: "migrate",
			env:     map[string]string{"MSP_DATABASE_URL": "postgres://msp_app@db/msp"},
			wantErr: true,
		},
		{
			name:    "migrate url of only spaces is missing",
			command: "migrate",
			env:     map[string]string{"MSP_MIGRATE_DATABASE_URL": "   "},
			wantErr: true,
		},
		{
			name:    "migrate with the migrate url",
			command: "migrate",
			env:     map[string]string{"MSP_MIGRATE_DATABASE_URL": "postgres://msp_owner@db/msp"},
			want: Config{
				Env: "local", LogLevel: slog.LevelInfo, HTTPAddr: ":8080",
				MigrateDatabaseURL: "postgres://msp_owner@db/msp",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse(tt.command, func(k string) string { return tt.env[k] })
			if (err != nil) != tt.wantErr {
				t.Fatalf("parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseTestEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want TestEnv
	}{
		{name: "unset", env: map[string]string{}, want: TestEnv{}},
		{
			name: "database and ci",
			env:  map[string]string{"MSP_TEST_DATABASE_URL": " postgres://postgres@localhost/postgres ", "CI": "true"},
			want: TestEnv{DatabaseURL: "postgres://postgres@localhost/postgres", CI: true},
		},
		{name: "ci other than true", env: map[string]string{"CI": "1"}, want: TestEnv{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTestEnv(func(k string) string { return tt.env[k] }); got != tt.want {
				t.Fatalf("parseTestEnv() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
