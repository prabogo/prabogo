package rabbitmq

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func setEnv(t *testing.T, values map[string]string) {
	t.Helper()
	for _, key := range []string{
		envHost, envPort, envUser, envPassword, envVHost,
		envTLS, envTLSSkip, envServerNm,
	} {
		t.Setenv(key, "")
	}
	for k, v := range values {
		t.Setenv(k, v)
	}
}

func TestBuildURLEncodePercentEncodesCredentials(t *testing.T) {
	raw := buildURLEncode("amqps", "user/name", "p@ss?word#1", "broker.example.com", "5671", "v/host?a=1")

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("generated uri is not parseable: %v", err)
	}
	if u.Scheme != "amqps" {
		t.Errorf("scheme = %q, want amqps", u.Scheme)
	}
	if u.Host != "broker.example.com:5671" {
		t.Errorf("host = %q", u.Host)
	}
	gotUser := u.User.Username()
	gotPass, _ := u.User.Password()
	if gotUser != "user/name" {
		t.Errorf("username = %q, want %q", gotUser, "user/name")
	}
	if gotPass != "p@ss?word#1" {
		t.Errorf("password = %q, want %q", gotPass, "p@ss?word#1")
	}
	if u.RawQuery != "" {
		t.Errorf("query = %q, want empty (vhost must not leak into query params)", u.RawQuery)
	}

	parsed, err := amqp.ParseURI(raw)
	if err != nil {
		t.Fatalf("ParseURI: %v", err)
	}
	if parsed.Vhost != "v/host?a=1" {
		t.Errorf("parsed vhost = %q, want %q", parsed.Vhost, "v/host?a=1")
	}
	if parsed.Username != "user/name" || parsed.Password != "p@ss?word#1" {
		t.Errorf("parsed credentials = %q/%q", parsed.Username, parsed.Password)
	}
}

func TestBuildURLEncodeCannotInjectTLSTypeParameters(t *testing.T) {
	raw := buildURLEncode("amqps", "u", "x?certfile=/etc/shadow&keyfile=/tmp/k", "host", "5671", "")

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("generated uri is not parseable: %v", err)
	}
	if u.RawQuery != "" {
		t.Fatalf("injected query params survived: %q", u.RawQuery)
	}
	pw, _ := u.User.Password()
	if pw != "x?certfile=/etc/shadow&keyfile=/tmp/k" {
		t.Errorf("password = %q, want the literal escaped value", pw)
	}
}

func TestResolveConnectionConfigDefaults(t *testing.T) {
	setEnv(t, map[string]string{
		envHost:     "rabbit",
		envPort:     "5672",
		envUser:     "app",
		envPassword: "s3cret",
	})

	cfg, err := resolveConnectionConfig()
	if err != nil {
		t.Fatalf("resolveConnectionConfig: %v", err)
	}
	if cfg.TLS {
		t.Error("TLS must be off unless MESSAGE_TLS is set")
	}
	if !strings.HasPrefix(cfg.URL, "amqp://app:") {
		t.Errorf("url = %q, want amqp scheme with user", cfg.URL)
	}
	if !strings.Contains(cfg.URL, "@rabbit:5672") {
		t.Errorf("url = %q, want host:port", cfg.URL)
	}
}

func TestResolveConnectionConfigTLSSwitchToDefaultTLSPort(t *testing.T) {
	setEnv(t, map[string]string{
		envHost:     "rabbit",
		envPort:     "5672",
		envUser:     "app",
		envPassword: "s3cret",
		envTLS:      "true",
	})

	cfg, err := resolveConnectionConfig()
	if err != nil {
		t.Fatalf("resolveConnectionConfig: %v", err)
	}
	if !strings.HasPrefix(cfg.URL, "amqps://") {
		t.Errorf("url = %q, want amqps scheme", cfg.URL)
	}
	if !strings.Contains(cfg.URL, "@rabbit:5671") {
		t.Errorf("url = %q, want the amqps default port when TLS is enabled", cfg.URL)
	}
	if cfg.tlsConfig().MinVersion != tls.VersionTLS12 {
		t.Errorf("min version = %d, want tls.VersionTLS12", cfg.tlsConfig().MinVersion)
	}
}

func TestResolveConnectionConfigExplicitPortKeptWithTLS(t *testing.T) {
	setEnv(t, map[string]string{
		envHost:     "rabbit",
		envPort:     "55671",
		envUser:     "app",
		envPassword: "s3cret",
		envTLS:      "true",
	})

	cfg, err := resolveConnectionConfig()
	if err != nil {
		t.Fatalf("resolveConnectionConfig: %v", err)
	}
	if !strings.Contains(cfg.URL, "@rabbit:55671") {
		t.Errorf("url = %q, want the explicitly configured port", cfg.URL)
	}
}

func TestResolveConnectionConfigServerNameAndSkipVerify(t *testing.T) {
	setEnv(t, map[string]string{
		envHost:     "rabbit",
		envUser:     "app",
		envPassword: "s3cret",
		envTLS:      "true",
		envTLSSkip:  "true",
		envServerNm: "broker.internal",
	})

	cfg, err := resolveConnectionConfig()
	if err != nil {
		t.Fatalf("resolveConnectionConfig: %v", err)
	}
	tlsCfg := cfg.tlsConfig()
	if tlsCfg.ServerName != "broker.internal" {
		t.Errorf("server name = %q", tlsCfg.ServerName)
	}
	if !tlsCfg.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should follow MESSAGE_TLS_SKIP_VERIFY")
	}
}

func TestResolveConnectionConfigRejectsBadInput(t *testing.T) {
	tests := []struct {
		name  string
		env   map[string]string
		exact bool
	}{
		{"missing host", map[string]string{envUser: "a", envPassword: "b"}, false},
		{"non numeric port", map[string]string{envHost: "h", envPort: "abc", envUser: "a", envPassword: "b"}, false},
		{"port out of range", map[string]string{envHost: "h", envPort: "70000", envUser: "a", envPassword: "b"}, false},
		{"missing credentials", map[string]string{envHost: "h"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, tc.env)
			if _, err := resolveConnectionConfig(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestResolveConnectionConfigVHostMatchesLegacyBehaviour(t *testing.T) {
	legacy := func(vhost string) string {
		return fmt.Sprintf("amqp://app:p@h:5672/%s", vhost)
	}
	for _, vhost := range []string{"", "myhost", "/myhost"} {
		setEnv(t, map[string]string{
			envHost: "h", envPort: "5672", envUser: "app", envPassword: "p",
			envVHost: vhost,
		})
		cfg, err := resolveConnectionConfig()
		if err != nil {
			t.Fatalf("vhost %q: resolveConnectionConfig: %v", vhost, err)
		}
		want, wantErr := amqp.ParseURI(legacy(vhost))
		got, gotErr := amqp.ParseURI(cfg.URL)
		if wantErr != nil || gotErr != nil {
			t.Fatalf("vhost %q: parse errors want=%v got=%v", vhost, wantErr, gotErr)
		}
		if got.Vhost != want.Vhost {
			t.Errorf("vhost %q: parsed %q, legacy produced %q", vhost, got.Vhost, want.Vhost)
		}
	}
}

func TestRedactURLHidesPassword(t *testing.T) {
	raw := "amqp://app:hunter2@rabbit:5672/vhost"
	got := redactURL(raw)
	if strings.Contains(got, "hunter2") {
		t.Errorf("redactURL leaked the password: %q", got)
	}
	if !strings.Contains(got, "app") {
		t.Errorf("redactURL should keep the username for diagnostics: %q", got)
	}
	if redactURL("not a url at all") != "<redacted>" {
		t.Error("unparseable urls must be fully redacted")
	}
}

func TestStripSecretRemovesCredentialsFromError(t *testing.T) {
	raw := "amqp://app:hunter2@rabbit:5672/"
	err := stripSecret(errors.New("dial error: amqp: connection refused for amqp://app:hunter2@rabbit:5672/"), raw)
	if strings.Contains(err.Error(), "hunter2") {
		t.Errorf("error still contains the password: %q", err.Error())
	}
	if strings.Contains(err.Error(), "app") {
		t.Errorf("error still contains the username: %q", err.Error())
	}
}

func TestAmqpConfigBoundsResources(t *testing.T) {
	setEnv(t, map[string]string{envHost: "h", envUser: "a", envPassword: "b", envTLS: "true"})
	cfg, err := resolveConnectionConfig()
	if err != nil {
		t.Fatalf("resolveConnectionConfig: %v", err)
	}
	amqpCfg := cfg.amqpConfig()
	if amqpCfg.FrameSize != DefaultMaxFramesize {
		t.Errorf("frame size = %d, want %d", amqpCfg.FrameSize, DefaultMaxFramesize)
	}
	if amqpCfg.TLSClientConfig == nil {
		t.Fatal("TLS config must be supplied explicitly so URI tls params are ignored")
	}
	if amqpCfg.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Error("TLS min version must be TLS 1.2")
	}
}

func TestAmqpConfigNoTLSConfigForPlaintext(t *testing.T) {
	setEnv(t, map[string]string{envHost: "h", envUser: "a", envPassword: "b"})
	cfg, err := resolveConnectionConfig()
	if err != nil {
		t.Fatalf("resolveConnectionConfig: %v", err)
	}
	if cfg.amqpConfig().TLSClientConfig != nil {
		t.Error("plaintext connections must not carry a TLS config")
	}
}

func TestValidatePrefetch(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		size    int
		wantErr bool
	}{
		{"zero is allowed", 0, 0, false},
		{"typical", 16, 0, false},
		{"max count", MaxPrefetchCount, 0, false},
		{"negative count rejected", -1, 0, true},
		{"huge count rejected", MaxPrefetchCount + 1, 0, true},
		{"overflowing count rejected", 1 << 31, 0, true},
		{"negative size rejected", 0, -1, true},
		{"huge size rejected", 0, MaxPrefetchSize + 1, true},
		{"max size allowed", 0, MaxPrefetchSize, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePrefetch(tc.count, tc.size)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validatePrefetch(%d,%d) = %v, wantErr %v", tc.count, tc.size, err, tc.wantErr)
			}
		})
	}
}
