package config

import (
	"log/slog"
	"testing"
)

// apiEnv is the minimal environment of `msp api`, merged with extra.
func apiEnv(extra map[string]string) map[string]string {
	env := map[string]string{
		"MSP_DATABASE_URL":      "postgres://msp_app@db/msp",
		"MSP_PUBLIC_URL":        "https://app.example.com",
		"MSP_KRATOS_PUBLIC_URL": "http://kratos:4433",
	}
	for k, v := range extra {
		env[k] = v
	}
	return env
}

// apiConfig is the Config of apiEnv(nil).
func apiConfig() Config {
	return Config{
		Env: "local", LogLevel: slog.LevelInfo, HTTPAddr: ":8080",
		DatabaseURL: "postgres://msp_app@db/msp", PublicURL: "https://app.example.com", KratosPublicURL: "http://kratos:4433",
	}
}

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
			env:     apiEnv(nil),
			want:    apiConfig(),
		},
		{
			name:    "explicit values",
			command: "api",
			env: map[string]string{
				"MSP_ENV": "production", "MSP_LOG_LEVEL": "debug", "MSP_HTTP_ADDR": ":9000",
				"MSP_DATABASE_URL":         "postgres://msp_app@db/msp",
				"MSP_MIGRATE_DATABASE_URL": " postgres://msp_owner@db/msp ",
				"MSP_PUBLIC_URL":           " https://App.Example.com/ ",
				"MSP_KRATOS_PUBLIC_URL":    "http://kratos:4433/",
			},
			want: Config{
				Env: "production", LogLevel: slog.LevelDebug, HTTPAddr: ":9000",
				DatabaseURL:        "postgres://msp_app@db/msp",
				MigrateDatabaseURL: "postgres://msp_owner@db/msp",
				PublicURL:          "https://app.example.com",
				KratosPublicURL:    "http://kratos:4433",
			},
		},
		{
			name:    "local public url with port",
			command: "api",
			env:     apiEnv(map[string]string{"MSP_PUBLIC_URL": "http://localhost:8000"}),
			want: func() Config {
				c := apiConfig()
				c.PublicURL = "http://localhost:8000"
				return c
			}(),
		},
		{name: "api needs the public url", command: "api", env: apiEnv(map[string]string{"MSP_PUBLIC_URL": ""}), wantErr: true},
		{name: "public url without scheme", command: "api", env: apiEnv(map[string]string{"MSP_PUBLIC_URL": "app.example.com"}), wantErr: true},
		{name: "public url not http", command: "api", env: apiEnv(map[string]string{"MSP_PUBLIC_URL": "ftp://app.example.com"}), wantErr: true},
		{name: "public url with a path", command: "api", env: apiEnv(map[string]string{"MSP_PUBLIC_URL": "https://example.com/app"}), wantErr: true},
		{name: "public url with a query", command: "api", env: apiEnv(map[string]string{"MSP_PUBLIC_URL": "https://example.com/?a=1"}), wantErr: true},
		{name: "api needs the kratos url", command: "api", env: apiEnv(map[string]string{"MSP_KRATOS_PUBLIC_URL": " "}), wantErr: true},
		{name: "kratos url without scheme", command: "api", env: apiEnv(map[string]string{"MSP_KRATOS_PUBLIC_URL": "kratos:4433"}), wantErr: true},
		{
			name:    "unknown env",
			command: "api",
			env:     apiEnv(map[string]string{"MSP_ENV": "prod"}),
			wantErr: true,
		},
		{
			name:    "unknown log level",
			command: "api",
			env:     apiEnv(map[string]string{"MSP_LOG_LEVEL": "loud"}),
			wantErr: true,
		},
		{
			name:    "api needs the database url",
			command: "api",
			env:     apiEnv(map[string]string{"MSP_DATABASE_URL": "", "MSP_MIGRATE_DATABASE_URL": "postgres://msp_owner@db/msp"}),
			wantErr: true,
		},
		{
			name:    "api database url of only spaces is missing",
			command: "api",
			env:     apiEnv(map[string]string{"MSP_DATABASE_URL": "   "}),
			wantErr: true,
		},
		{
			name:    "worker does not need api settings yet",
			command: "worker",
			env:     map[string]string{},
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
