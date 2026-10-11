package rabbitmq

import (
	"context"
	"strings"
	"testing"
)

func validCallback() func(context.Context, []byte) bool {
	return func(context.Context, []byte) bool { return true }
}

func baseSubscriberConfig() SubscriberConfig {
	return SubscriberConfig{
		Exchange:     "exchange",
		ExchangeKind: KindFanOut,
		Queue:        "queue",
		RouteKey:     "key",
		Callback:     validCallback(),
	}
}

func TestSubscriberConfigValidateAcceptsDefaults(t *testing.T) {
	cfg := baseSubscriberConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config rejected: %v", err)
	}
}

func TestSubscriberConfigValidateRejectsBadLimits(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*SubscriberConfig)
	}{
		{"negative prefetch count", func(c *SubscriberConfig) { c.PrefetchCount = -1 }},
		{"oversized prefetch count", func(c *SubscriberConfig) { c.PrefetchCount = MaxPrefetchCount + 1 }},
		{"signed overflow prefetch count", func(c *SubscriberConfig) { c.PrefetchCount = 1 << 31 }},
		{"negative prefetch size", func(c *SubscriberConfig) { c.PrefetchSize = -1 }},
		{"oversized prefetch size", func(c *SubscriberConfig) { c.PrefetchSize = MaxPrefetchSize + 1 }},
		{"negative max body", func(c *SubscriberConfig) { c.MaxBodyBytes = -2 }},
		{"empty exchange", func(c *SubscriberConfig) { c.Exchange = "" }},
		{"empty kind", func(c *SubscriberConfig) { c.ExchangeKind = "" }},
		{"empty queue", func(c *SubscriberConfig) { c.Queue = "" }},
		{"nil callback", func(c *SubscriberConfig) { c.Callback = nil }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := baseSubscriberConfig()
			tc.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestSubscriberConfigPrefetchDefaults(t *testing.T) {
	cfg := baseSubscriberConfig()
	if got := cfg.prefetchCount(); got != DefaultPrefetchCount {
		t.Errorf("prefetchCount = %d, want the bounded default %d", got, DefaultPrefetchCount)
	}
	cfg.PrefetchCount = 3
	if got := cfg.prefetchCount(); got != 3 {
		t.Errorf("prefetchCount = %d, want 3", got)
	}
}

func TestSubscriberConfigMaxBodyLimit(t *testing.T) {
	cfg := baseSubscriberConfig()
	if got := cfg.maxBodyLimit(); got != DefaultMaxBodyBytes {
		t.Errorf("maxBodyLimit = %d, want default %d", got, DefaultMaxBodyBytes)
	}
	cfg.MaxBodyBytes = 1024
	if got := cfg.maxBodyLimit(); got != 1024 {
		t.Errorf("maxBodyLimit = %d, want 1024", got)
	}
	cfg.MaxBodyBytes = -1
	if got := cfg.maxBodyLimit(); got != 0 {
		t.Errorf("maxBodyLimit = %d, want 0 (unlimited)", got)
	}
}

func TestSubscriberWithConfigRejectsUnsafePrefetch(t *testing.T) {
	cfg := baseSubscriberConfig()
	cfg.PrefetchCount = -1
	err := SubscriberWithConfig(cfg)
	if err == nil {
		t.Fatal("expected an error for a negative prefetch count")
	}
	if !strings.Contains(err.Error(), "prefetch") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSubscriberWithConfigValidatesBeforeDialling(t *testing.T) {
	t.Setenv(envHost, "")
	cfg := baseSubscriberConfig()
	cfg.Callback = nil
	if err := SubscriberWithConfig(cfg); err == nil {
		t.Fatal("expected a validation error before any dial attempt")
	}
}
