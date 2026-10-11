package rabbitmq

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Bounds against the broker-controlled DoS classes reported for amqp091-go.
const (
	DefaultMaxBodyBytes = 8 << 20 // 8 MiB

	DefaultPrefetchCount = 16

	// MaxPrefetchCount is the octet upper bound for PrefetchCount.
	MaxPrefetchCount = 255

	// MaxPrefetchSize is the byte upper bound for PrefetchSize.
	MaxPrefetchSize = 1 << 30 // 1 GiB

	// 0 means unlimited to the library, so a value is always proposed.
	DefaultMaxFramesize = 131072

	DefaultHeartbeat = 10 * time.Second
)

const (
	envHost     = "MESSAGE_HOST"
	envPort     = "MESSAGE_PORT"
	envUser     = "MESSAGE_USER"
	envPassword = "MESSAGE_PASSWORD"
	envVHost    = "MESSAGE_VHOST"
	envTLS      = "MESSAGE_TLS"
	envTLSSkip  = "MESSAGE_TLS_SKIP_VERIFY"
	envServerNm = "MESSAGE_TLS_SERVER_NAME"
)

var errNoCredentials = errors.New("rabbitmq credentials missing")

type connectionConfig struct {
	URL                string
	TLS                bool
	InsecureSkipVerify bool
	ServerName         string
}

// buildURLEncode percent-encodes credentials and vhost so reserved characters
// cannot inject URI components such as certfile/keyfile (CVE-2026-77404).
func buildURLEncode(scheme, user, password, host, port, vhost string) string {
	u := url.URL{
		Scheme: scheme,
		Host:   host + ":" + port,
	}
	if user != "" || password != "" {
		u.User = url.UserPassword(user, password)
	}
	if vhost != "" {
		u.Path = "/" + vhost
	}
	return u.String()
}

func resolveConnectionConfig() (connectionConfig, error) {
	host := strings.TrimSpace(os.Getenv(envHost))
	if host == "" {
		return connectionConfig{}, errors.New("rabbitmq host is not configured")
	}

	port := strings.TrimSpace(os.Getenv(envPort))
	if port == "" {
		port = "5672"
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return connectionConfig{}, fmt.Errorf("invalid rabbitmq port %q", port)
	}

	user := os.Getenv(envUser)
	password := os.Getenv(envPassword)
	if user == "" && password == "" {
		return connectionConfig{}, errNoCredentials
	}

	vhost := os.Getenv(envVHost)

	useTLS := envFlag(envTLS)
	if useTLS && port == "5672" {
		port = "5671"
	}

	scheme := "amqp"
	if useTLS {
		scheme = "amqps"
	}

	return connectionConfig{
		URL:                buildURLEncode(scheme, user, password, host, port, vhost),
		TLS:                useTLS,
		InsecureSkipVerify: useTLS && envFlag(envTLSSkip),
		ServerName:         strings.TrimSpace(os.Getenv(envServerNm)),
	}, nil
}

func envFlag(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "true" || v == "1" || v == "yes" || v == "on"
}

// tlsConfig enforces TLS 1.2+ (CVE-2026-77405).
func (c connectionConfig) tlsConfig() *tls.Config {
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: c.InsecureSkipVerify,
	}
	if c.ServerName != "" {
		cfg.ServerName = c.ServerName
	}
	return cfg
}

// amqpConfig supplies TLSClientConfig so URI TLS params (certfile/keyfile/
// cacertfile) are ignored, and bounds the resources the broker may request.
func (c connectionConfig) amqpConfig() amqp.Config {
	return amqp.Config{
		TLSClientConfig: c.tlsConfigOrNil(),
		FrameSize:       DefaultMaxFramesize,
		Heartbeat:       DefaultHeartbeat,
	}
}

func (c connectionConfig) tlsConfigOrNil() *tls.Config {
	if !c.TLS {
		return nil
	}
	return c.tlsConfig()
}

// redactURL removes the userinfo from an AMQP URI so it is safe to log.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return "<redacted>"
	}
	if _, hasPassword := u.User.Password(); hasPassword {
		u.User = url.User(u.User.Username())
	} else {
		u.User = url.User("user")
	}
	return u.String()
}

// dial opens a connection with credentials stripped from any error.
func dial() (*amqp.Connection, error) {
	cfg, err := resolveConnectionConfig()
	if err != nil {
		return nil, err
	}

	conn, err := amqp.DialConfig(cfg.URL, cfg.amqpConfig())
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial %s failed: %w", redactURL(cfg.URL), stripSecret(err, cfg.URL))
	}
	return conn, nil
}

// stripSecret removes credential material the library may echo into an error.
func stripSecret(err error, rawURL string) error {
	msg := err.Error()
	if u, parseErr := url.Parse(rawURL); parseErr == nil && u.User != nil {
		if pw, ok := u.User.Password(); ok && pw != "" {
			msg = strings.ReplaceAll(msg, pw, "<redacted>")
		}
		if name := u.User.Username(); name != "" {
			msg = strings.ReplaceAll(msg, name, "<redacted>")
		}
	}
	return errors.New(msg)
}

// validatePrefetch rejects values that cannot be safely converted to the
// unsigned QoS fields (CVE-2026-77406).
func validatePrefetch(count, size int) error {
	if count < 0 || count > MaxPrefetchCount {
		return fmt.Errorf("prefetch count %d out of range [0,%d]", count, MaxPrefetchCount)
	}
	if size < 0 || size > MaxPrefetchSize {
		return fmt.Errorf("prefetch size %d out of range [0,%d]", size, MaxPrefetchSize)
	}
	return nil
}
