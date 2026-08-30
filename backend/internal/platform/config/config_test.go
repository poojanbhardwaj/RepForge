package config

import (
	"strings"
	"testing"
)

func clearPostgresRedirectEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"PGHOST",
		"PGPORT",
		"PGSERVICE",
		"PGSERVICEFILE",
		"PGSSLMODE",
		"PGTARGETSESSIONATTRS",
	} {
		t.Setenv(name, "")
	}
}

func lookup(values map[string]string) Lookup {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}

func validValues() map[string]string {
	return map[string]string{
		"APP_ENV":          "local",
		"AUTH_MODE":        "dev",
		"HTTP_ADDRESS":     "127.0.0.1:8080",
		"DATABASE_URL":     "postgres://127.0.0.1/test",
		"REDIS_URL":        "redis://127.0.0.1:6379/0",
		"S3_ENDPOINT":      "http://[::1]:3900",
		"DEV_AUTH_TOKEN":   strings.Repeat("x", 24),
		"DEV_AUTH_SUBJECT": "local-user",
	}
}

func TestLoadAcceptsLocalDevelopmentAuth(t *testing.T) {
	clearPostgresRedirectEnvironment(t)
	if _, err := LoadWithLookup(lookup(validValues())); err != nil {
		t.Fatalf("LoadWithLookup() error = %v", err)
	}
}

func TestLoadAcceptsOIDCAndRejectsIncompleteOrInsecureOIDC(t *testing.T) {
	clearPostgresRedirectEnvironment(t)
	validOIDC := validValues()
	validOIDC["AUTH_MODE"] = "oidc"
	validOIDC["OIDC_ISSUER"] = "https://repforge-dev-poojan.jp.auth0.com/"
	validOIDC["OIDC_AUDIENCE"] = "https://api.dev.repforge.invalid"
	validOIDC["OIDC_MOBILE_CLIENT_ID"] = "mobile-client"
	validOIDC["OIDC_WEB_CLIENT_ID"] = "web-client"
	if _, err := LoadWithLookup(lookup(validOIDC)); err != nil {
		t.Fatalf("valid OIDC config: %v", err)
	}
	for name, mutate := range map[string]func(map[string]string){
		"http issuer":           func(values map[string]string) { values["OIDC_ISSUER"] = "http://issuer.invalid/" },
		"issuer user info":      func(values map[string]string) { values["OIDC_ISSUER"] = "https://user@issuer.invalid/" },
		"issuer path":           func(values map[string]string) { values["OIDC_ISSUER"] = "https://issuer.invalid/path/" },
		"missing audience":      func(values map[string]string) { values["OIDC_AUDIENCE"] = "" },
		"http audience":         func(values map[string]string) { values["OIDC_AUDIENCE"] = "http://api.invalid" },
		"missing mobile client": func(values map[string]string) { values["OIDC_MOBILE_CLIENT_ID"] = "" },
		"spaced mobile client":  func(values map[string]string) { values["OIDC_MOBILE_CLIENT_ID"] = " mobile-client" },
		"placeholder client": func(values map[string]string) {
			values["OIDC_MOBILE_CLIENT_ID"] = "__POPULATE_AUTH0_MOBILE_CLIENT_ID__"
		},
		"duplicate client":     func(values map[string]string) { values["OIDC_WEB_CLIENT_ID"] = "mobile-client" },
		"unsafe terms version": func(values map[string]string) { values["TERMS_VERSION"] = "draft terms" },
	} {
		t.Run(name, func(t *testing.T) {
			values := make(map[string]string, len(validOIDC))
			for key, value := range validOIDC {
				values[key] = value
			}
			mutate(values)
			if _, err := LoadWithLookup(lookup(values)); err == nil {
				t.Fatal("expected OIDC configuration rejection")
			}
		})
	}
}

func TestLoadRejectsDevelopmentAuthOutsideLocalOrTest(t *testing.T) {
	clearPostgresRedirectEnvironment(t)
	for _, environment := range []string{"staging", "production"} {
		t.Run(environment, func(t *testing.T) {
			values := validValues()
			values["APP_ENV"] = environment
			_, err := LoadWithLookup(lookup(values))
			if err == nil || !strings.Contains(err.Error(), "forbidden") {
				t.Fatalf("expected fail-closed error, got %v", err)
			}
		})
	}
}

func TestLoadRejectsShortDevelopmentToken(t *testing.T) {
	clearPostgresRedirectEnvironment(t)
	values := validValues()
	values["DEV_AUTH_TOKEN"] = "short"
	if _, err := LoadWithLookup(lookup(values)); err == nil {
		t.Fatal("expected short token rejection")
	}
}

func TestLoadRejectsNonLoopbackLocalDependencies(t *testing.T) {
	clearPostgresRedirectEnvironment(t)
	tests := map[string]struct {
		name  string
		value string
	}{
		"wildcard HTTP bind": {name: "HTTP_ADDRESS", value: "0.0.0.0:8080"},
		"remote database":    {name: "DATABASE_URL", value: "postgres://db.example.test/repforge"},
		"remote Redis":       {name: "REDIS_URL", value: "rediss://cache.example.test/0"},
		"remote S3":          {name: "S3_ENDPOINT", value: "https://objects.example.test"},
	}
	for testName, testCase := range tests {
		t.Run(testName, func(t *testing.T) {
			values := validValues()
			values[testCase.name] = testCase.value
			_, err := LoadWithLookup(lookup(values))
			if err == nil || !strings.Contains(err.Error(), testCase.name) {
				t.Fatalf("expected %s loopback rejection, got %v", testCase.name, err)
			}
		})
	}
}

func TestLoadAcceptsExplicitIPv4AndIPv6Loopback(t *testing.T) {
	clearPostgresRedirectEnvironment(t)
	for _, databaseURL := range []string{
		"postgres://127.0.0.1/repforge",
		"postgresql://127.12.0.1/repforge",
		"postgres://[::1]/repforge",
	} {
		t.Run(databaseURL, func(t *testing.T) {
			values := validValues()
			values["DATABASE_URL"] = databaseURL
			if _, err := LoadWithLookup(lookup(values)); err != nil {
				t.Fatalf("LoadWithLookup() error = %v", err)
			}
		})
	}
}

func TestLoadRejectsHostnameEvenWhenNamedLocalhost(t *testing.T) {
	clearPostgresRedirectEnvironment(t)
	values := validValues()
	values["DATABASE_URL"] = "postgres://localhost/repforge"
	if _, err := LoadWithLookup(lookup(values)); err == nil {
		t.Fatal("expected hostname rejection; a literal loopback IP is required")
	}
}

func TestParseDatabaseConfigRejectsRedirectsAndNonLoopbackTargets(t *testing.T) {
	clearPostgresRedirectEnvironment(t)
	tests := map[string]string{
		"query host override": "postgres://127.0.0.1/repforge?host=127.0.0.2",
		"query port override": "postgres://127.0.0.1:5432/repforge?port=5433",
		"remote primary host": "postgres://192.0.2.10/repforge?sslmode=disable",
		"mixed URL hosts":     "postgres://127.0.0.1:5432,192.0.2.10:5432/repforge?sslmode=disable",
		"remote fallback host": "host=127.0.0.1,198.51.100.20 port=5432,5433 " +
			"dbname=repforge sslmode=disable",
		"service option": "host=127.0.0.1 dbname=repforge service=redirect",
		"service file option": "host=127.0.0.1 dbname=repforge " +
			"servicefile=missing.conf",
	}
	for testName, databaseURL := range tests {
		t.Run(testName, func(t *testing.T) {
			if _, err := ParseDatabaseConfig(databaseURL); err == nil {
				t.Fatalf("ParseDatabaseConfig(%q) unexpectedly succeeded", databaseURL)
			}
		})
	}
}

func TestParseDatabaseConfigRejectsMalformedDSNs(t *testing.T) {
	clearPostgresRedirectEnvironment(t)
	for _, databaseURL := range []string{
		"postgres://%zz",
		"postgres://127.0.0.1:not-a-port/repforge",
		"host='unterminated",
		"not-a-dsn",
	} {
		t.Run(databaseURL, func(t *testing.T) {
			if _, err := ParseDatabaseConfig(databaseURL); err == nil {
				t.Fatalf("ParseDatabaseConfig(%q) unexpectedly succeeded", databaseURL)
			}
		})
	}
}

func TestParseDatabaseConfigAcceptsEveryLiteralLoopbackTarget(t *testing.T) {
	clearPostgresRedirectEnvironment(t)
	for _, databaseURL := range []string{
		"postgres://127.0.0.1:5432/repforge?sslmode=disable",
		"postgresql://[::1]:5432/repforge?sslmode=require",
		"host=127.0.0.1,127.20.30.40 port=5432,5433 dbname=repforge sslmode=disable",
		"host=::1 dbname=repforge sslmode=disable application_name=repforge-test",
	} {
		t.Run(databaseURL, func(t *testing.T) {
			if _, err := ParseDatabaseConfig(databaseURL); err != nil {
				t.Fatalf("ParseDatabaseConfig(%q) error = %v", databaseURL, err)
			}
		})
	}
}

func TestParseDatabaseConfigValidatesEffectiveEnvironmentHost(t *testing.T) {
	clearPostgresRedirectEnvironment(t)
	t.Setenv("PGHOST", "203.0.113.10")
	if _, err := ParseDatabaseConfig("dbname=repforge sslmode=disable"); err == nil {
		t.Fatal("expected effective PGHOST rejection")
	}
}
