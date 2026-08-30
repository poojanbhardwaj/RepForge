package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Environment string

const (
	EnvironmentLocal      Environment = "local"
	EnvironmentTest       Environment = "test"
	EnvironmentStaging    Environment = "staging"
	EnvironmentProduction Environment = "production"
)

type Config struct {
	Environment       Environment
	AuthMode          string
	HTTPAddress       string
	LogLevel          slog.Level
	DatabaseURL       string
	RedisURL          string
	S3Endpoint        string
	S3Region          string
	S3AccessKey       string
	S3SecretKey       string
	S3Bucket          string
	S3ForcePathStyle  bool
	DevAuthToken      string
	DevAuthSubject    string
	DevAuthProvider   string
	OIDCIssuer        string
	OIDCAudience      string
	OIDCClientIDs     []string
	CORSAllowedOrigin string
	TermsVersion      string
	PrivacyVersion    string
	ShutdownTimeout   time.Duration
	ReadinessTimeout  time.Duration
	WorkerIdleAllowed bool
}

type Lookup func(string) (string, bool)

func Load() (Config, error) {
	return LoadWithLookup(os.LookupEnv)
}

func LoadWithLookup(lookup Lookup) (Config, error) {
	value := func(name, fallback string) string {
		if result, ok := lookup(name); ok {
			return strings.TrimSpace(result)
		}
		return fallback
	}
	exactValue := func(name, fallback string) string {
		if result, ok := lookup(name); ok {
			return result
		}
		return fallback
	}
	parseDuration := func(name, fallback string) (time.Duration, error) {
		result, err := time.ParseDuration(value(name, fallback))
		if err != nil || result <= 0 {
			return 0, fmt.Errorf("%s must be a positive duration", name)
		}
		return result, nil
	}
	environment := Environment(value("APP_ENV", ""))
	localCORSFallback := ""
	localDocumentFallback := ""
	if environment == EnvironmentLocal || environment == EnvironmentTest {
		localCORSFallback = "http://127.0.0.1:3000"
		localDocumentFallback = "draft-local-1"
	}

	shutdown, err := parseDuration("SHUTDOWN_TIMEOUT", "10s")
	if err != nil {
		return Config{}, err
	}
	readiness, err := parseDuration("READINESS_TIMEOUT", "2s")
	if err != nil {
		return Config{}, err
	}
	level := new(slog.LevelVar)
	if err := level.UnmarshalText([]byte(value("LOG_LEVEL", "info"))); err != nil {
		return Config{}, errors.New("LOG_LEVEL must be debug, info, warn, or error")
	}

	cfg := Config{
		Environment:      environment,
		AuthMode:         value("AUTH_MODE", ""),
		HTTPAddress:      value("HTTP_ADDRESS", "127.0.0.1:8080"),
		LogLevel:         level.Level(),
		DatabaseURL:      value("DATABASE_URL", ""),
		RedisURL:         value("REDIS_URL", ""),
		S3Endpoint:       value("S3_ENDPOINT", ""),
		S3Region:         value("S3_REGION", ""),
		S3AccessKey:      value("GARAGE_ACCESS_KEY", ""),
		S3SecretKey:      value("GARAGE_SECRET_KEY", ""),
		S3Bucket:         value("GARAGE_BUCKET", ""),
		S3ForcePathStyle: strings.EqualFold(value("S3_FORCE_PATH_STYLE", "false"), "true"),
		DevAuthToken:     value("DEV_AUTH_TOKEN", ""),
		DevAuthSubject:   value("DEV_AUTH_SUBJECT", ""),
		DevAuthProvider:  value("DEV_AUTH_PROVIDER", "repforge-dev"),
		OIDCIssuer:       exactValue("OIDC_ISSUER", ""),
		OIDCAudience:     exactValue("OIDC_AUDIENCE", ""),
		OIDCClientIDs: []string{
			exactValue("OIDC_MOBILE_CLIENT_ID", ""),
			exactValue("OIDC_WEB_CLIENT_ID", ""),
		},
		CORSAllowedOrigin: exactValue("CORS_ALLOWED_ORIGIN", localCORSFallback),
		TermsVersion:      exactValue("TERMS_VERSION", localDocumentFallback),
		PrivacyVersion:    exactValue("PRIVACY_VERSION", localDocumentFallback),
		ShutdownTimeout:   shutdown,
		ReadinessTimeout:  readiness,
		WorkerIdleAllowed: strings.EqualFold(value("WORKER_IDLE_ALLOWED", "false"), "true"),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	switch c.Environment {
	case EnvironmentLocal, EnvironmentTest, EnvironmentStaging, EnvironmentProduction:
	default:
		return errors.New("APP_ENV must be local, test, staging, or production")
	}
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	switch c.AuthMode {
	case "dev":
		if c.Environment != EnvironmentLocal && c.Environment != EnvironmentTest {
			return errors.New("development authentication is forbidden outside local/test")
		}
		if len(c.DevAuthToken) < 16 || c.DevAuthSubject == "" || c.DevAuthProvider == "" {
			return errors.New("DEV_AUTH_TOKEN (16+ characters), DEV_AUTH_SUBJECT, and DEV_AUTH_PROVIDER are required")
		}
	case "oidc":
		if err := validateOIDC(c.OIDCIssuer, c.OIDCAudience, c.OIDCClientIDs); err != nil {
			return err
		}
	default:
		return errors.New("AUTH_MODE must be dev or oidc")
	}
	if !validVersionIdentifier(c.TermsVersion) {
		return errors.New("TERMS_VERSION must be 1 to 80 safe ASCII characters")
	}
	if !validVersionIdentifier(c.PrivacyVersion) {
		return errors.New("PRIVACY_VERSION must be 1 to 80 safe ASCII characters")
	}
	if c.Environment == EnvironmentLocal || c.Environment == EnvironmentTest {
		if c.CORSAllowedOrigin != "http://127.0.0.1:3000" {
			return errors.New("CORS_ALLOWED_ORIGIN must be exactly http://127.0.0.1:3000 in local/test")
		}
	} else if c.CORSAllowedOrigin != "" {
		return errors.New("CORS_ALLOWED_ORIGIN must be empty outside local/test until a production origin is approved")
	}
	if err := validateLoopbackAddress("HTTP_ADDRESS", c.HTTPAddress); err != nil {
		return err
	}
	if _, err := ParseDatabaseConfig(c.DatabaseURL); err != nil {
		return err
	}
	if c.RedisURL == "" {
		return errors.New("REDIS_URL is required")
	}
	if err := validateLoopbackURL("REDIS_URL", c.RedisURL, "redis", "rediss"); err != nil {
		return err
	}
	if c.S3Endpoint == "" {
		return errors.New("S3_ENDPOINT is required")
	}
	if err := validateLoopbackURL("S3_ENDPOINT", c.S3Endpoint, "http", "https"); err != nil {
		return err
	}
	return nil
}

func validateOIDC(issuer, audience string, clientIDs []string) error {
	parsed, err := url.Parse(issuer)
	if err != nil || len(issuer) > 80 || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Path != "/" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.String() != issuer || !strings.HasSuffix(issuer, "/") {
		return errors.New("OIDC_ISSUER must be a canonical HTTPS URL ending with a slash")
	}
	audienceURL, audienceErr := url.Parse(audience)
	if audienceErr != nil || audience == "" || len(audience) > 255 || audienceURL.Scheme != "https" || audienceURL.Host == "" || audienceURL.User != nil || audienceURL.RawQuery != "" || audienceURL.Fragment != "" || audienceURL.String() != audience {
		return errors.New("OIDC_AUDIENCE must be a canonical HTTPS identifier of at most 255 characters")
	}
	seen := make(map[string]struct{}, len(clientIDs))
	for _, clientID := range clientIDs {
		if clientID == "" || len(clientID) > 255 || strings.TrimSpace(clientID) != clientID || containsControl(clientID) || strings.HasPrefix(clientID, "__POPULATE_") {
			return errors.New("OIDC_MOBILE_CLIENT_ID and OIDC_WEB_CLIENT_ID are required")
		}
		if _, duplicate := seen[clientID]; duplicate {
			return errors.New("OIDC mobile and web client IDs must be distinct")
		}
		seen[clientID] = struct{}{}
	}
	return nil
}

func validVersionIdentifier(value string) bool {
	if len(value) < 1 || len(value) > 80 {
		return false
	}
	for _, character := range []byte(value) {
		if (character < 'A' || character > 'Z') && (character < 'a' || character > 'z') &&
			(character < '0' || character > '9') && !strings.ContainsRune("._:-", rune(character)) {
			return false
		}
	}
	return true
}

func containsControl(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return true
		}
	}
	return false
}

var allowedDatabaseDSNKeys = []string{
	"host",
	"port",
	"database",
	"user",
	"password",
	"connect_timeout",
	"sslmode",
	"sslnegotiation",
	"sslsni",
	"target_session_attrs",
	"application_name",
	"statement_cache_capacity",
	"description_cache_capacity",
	"default_query_exec_mode",
}

var databaseURLRedirectKeys = map[string]struct{}{
	"host":        {},
	"hostaddr":    {},
	"port":        {},
	"service":     {},
	"servicefile": {},
}

// ParseDatabaseConfig uses the pinned pgx parser and then validates every
// effective connection target. The connection-string allowlist also prevents
// pgx from reading service, password, or TLS files named by the DSN.
func ParseDatabaseConfig(raw string) (*pgx.ConnConfig, error) {
	parsed, err := pgx.ParseConfigWithOptions(raw, pgx.ParseConfigOptions{
		ParseConfigOptions: pgconn.ParseConfigOptions{
			ConnStringAllowedKeys: allowedDatabaseDSNKeys,
		},
	})
	if err != nil {
		return nil, errors.New("DATABASE_URL must be a valid supported PostgreSQL DSN")
	}
	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") {
		parsedURL, parseErr := url.Parse(raw)
		if parseErr != nil {
			return nil, errors.New("DATABASE_URL must be a valid supported PostgreSQL DSN")
		}
		for key := range parsedURL.Query() {
			if _, redirects := databaseURLRedirectKeys[strings.ToLower(key)]; redirects {
				return nil, fmt.Errorf("DATABASE_URL query option %q may redirect the connection", key)
			}
		}
	}
	if err := ValidateDatabaseConfig(parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

// RevalidateDatabaseConfig checks both the original DSN options and the
// effective targets after a parsed configuration has been copied or changed.
func RevalidateDatabaseConfig(parsed *pgx.ConnConfig) error {
	if parsed == nil {
		return errors.New("DATABASE_URL configuration is required")
	}
	if _, err := ParseDatabaseConfig(parsed.ConnString()); err != nil {
		return err
	}
	return ValidateDatabaseConfig(parsed)
}

// ValidateDatabaseConfig must be called after copying or changing a parsed
// config and immediately before using it for a database operation.
func ValidateDatabaseConfig(parsed *pgx.ConnConfig) error {
	if parsed == nil {
		return errors.New("DATABASE_URL configuration is required")
	}
	targets := make([]string, 0, len(parsed.Fallbacks)+1)
	targets = append(targets, parsed.Host)
	for _, fallback := range parsed.Fallbacks {
		if fallback == nil {
			return errors.New("DATABASE_URL contains an invalid fallback target")
		}
		targets = append(targets, fallback.Host)
	}
	for _, host := range targets {
		if !isLoopbackHost(host) {
			return fmt.Errorf("DATABASE_URL effective host %q must be a literal loopback IP", host)
		}
	}
	return nil
}

func validateLoopbackAddress(name, address string) error {
	host, port, err := net.SplitHostPort(address)
	portNumber, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || portNumber < 1 || portNumber > 65535 || !isLoopbackHost(host) {
		return fmt.Errorf("%s must bind to an explicit loopback IP and numeric port", name)
	}
	return nil
}

func validateLoopbackURL(name, raw string, allowedSchemes ...string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" || !isLoopbackHost(parsed.Hostname()) {
		return fmt.Errorf("%s must use an explicit loopback IP", name)
	}
	for _, scheme := range allowedSchemes {
		if parsed.Scheme == scheme {
			return nil
		}
	}
	return fmt.Errorf("%s uses an unsupported scheme", name)
}

func isLoopbackHost(host string) bool {
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}
