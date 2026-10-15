package main

import "testing"

func TestLoadConfigNormalizesOriginAndCookieSecurity(t *testing.T) {
	t.Setenv("LINKHUB_DATABASE_URL", "postgres://localhost/linkhub")
	for _, tc := range []struct {
		raw, normalized string
		secure          bool
	}{
		{" HTTPS://EXAMPLE.TEST:443/ ", "https://example.test", true},
		{"HTTP://LOCALHOST:80/", "http://localhost", false},
		{"http://127.0.0.1:8097", "http://127.0.0.1:8097", false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			t.Setenv("LINKHUB_ORIGIN", tc.raw)
			cfg, e := LoadConfig()
			if e != nil {
				t.Fatal(e)
			}
			if cfg.Origin != tc.normalized || cfg.SecureCookies != tc.secure {
				t.Fatalf("origin=%q secure=%v", cfg.Origin, cfg.SecureCookies)
			}
		})
	}
}

func TestLoadConfigRejectsNonOriginURL(t *testing.T) {
	t.Setenv("LINKHUB_DATABASE_URL", "postgres://localhost/linkhub")
	for _, raw := range []string{"https://user@example.test", "https://example.test/path", "https://example.test?x=1", "file:///tmp"} {
		t.Setenv("LINKHUB_ORIGIN", raw)
		if _, e := LoadConfig(); e == nil {
			t.Errorf("accepted origin %q", raw)
		}
	}
}
