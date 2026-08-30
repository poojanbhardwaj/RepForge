package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type oidcFixture struct {
	t                 *testing.T
	server            *httptest.Server
	issuer            string
	now               time.Time
	mu                sync.Mutex
	keys              map[string]*rsa.PublicKey
	discoveryRequests int
	jwksRequests      int
	failDiscovery     bool
	authenticator     *OIDCAuthenticator
}

func newOIDCFixture(t *testing.T) *oidcFixture {
	t.Helper()
	fixture := &oidcFixture{t: t, now: time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC), keys: make(map[string]*rsa.PublicKey)}
	fixture.server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			fixture.mu.Lock()
			fixture.discoveryRequests++
			fail := fixture.failDiscovery
			fixture.mu.Unlock()
			if fail {
				http.Error(response, "synthetic unavailable", http.StatusServiceUnavailable)
				return
			}
			response.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(response).Encode(map[string]string{"issuer": fixture.issuer, "jwks_uri": fixture.issuer + "jwks"})
		case "/jwks":
			response.Header().Set("Content-Type", "application/jwk-set+json")
			fixture.mu.Lock()
			fixture.jwksRequests++
			keys := make([]map[string]string, 0, len(fixture.keys))
			for kid, key := range fixture.keys {
				keys = append(keys, encodeTestJWK(kid, key))
			}
			fixture.mu.Unlock()
			_ = json.NewEncoder(response).Encode(map[string]any{"keys": keys})
		default:
			http.NotFound(response, request)
		}
	}))
	fixture.issuer = fixture.server.URL + "/"
	authenticator, err := NewOIDCAuthenticator(OIDCOptions{
		Issuer: fixture.issuer, Audience: "https://api.test.invalid", AllowedClientIDs: []string{"mobile-client", "web-client"},
		HTTPClient: fixture.server.Client(), CacheTTL: time.Hour, ClockSkew: 30 * time.Second,
		Now: func() time.Time { return fixture.now }, AllowInsecureIssuerInTest: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture.authenticator = authenticator
	t.Cleanup(fixture.server.Close)
	return fixture
}

func (fixture *oidcFixture) addKey(kid string, key *rsa.PrivateKey) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	fixture.keys[kid] = &key.PublicKey
}

func (fixture *oidcFixture) replaceKey(kid string, key *rsa.PrivateKey) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	fixture.keys = map[string]*rsa.PublicKey{kid: &key.PublicKey}
}

func (fixture *oidcFixture) requestCount() int {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return fixture.jwksRequests
}

func (fixture *oidcFixture) discoveryRequestCount() int {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return fixture.discoveryRequests
}

func encodeTestJWK(kid string, key *rsa.PublicKey) map[string]string {
	exponent := big.NewInt(int64(key.E)).Bytes()
	return map[string]string{
		"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(exponent),
	}
}

func testPrivateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func signAccessToken(t *testing.T, fixture *oidcFixture, key *rsa.PrivateKey, kid string, mutate func(*accessTokenClaims)) string {
	t.Helper()
	verified := true
	claims := accessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: fixture.issuer, Subject: "auth0|synthetic-user",
			Audience:  jwt.ClaimStrings{"https://api.test.invalid"},
			ExpiresAt: jwt.NewNumericDate(fixture.now.Add(15 * time.Minute)),
			NotBefore: jwt.NewNumericDate(fixture.now.Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(fixture.now.Add(-time.Minute)),
		},
		Scope: "openid profile:read profile:write", ClientID: "mobile-client", EmailVerified: &verified,
	}
	if mutate != nil {
		mutate(&claims)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	token.Header["typ"] = "at+jwt"
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestOIDCAuthenticatorValidatesAuth0RFC9068AccessTokenProfile(t *testing.T) {
	fixture := newOIDCFixture(t)
	key := testPrivateKey(t)
	fixture.addKey("current", key)
	principal, err := fixture.authenticator.Authenticate(context.Background(), signAccessToken(t, fixture, key, "current", nil))
	if err != nil {
		t.Fatal(err)
	}
	if principal.Provider != fixture.issuer || principal.Subject != "auth0|synthetic-user" || principal.ClientID != "mobile-client" {
		t.Fatalf("unexpected principal: %+v", principal)
	}
	if len(principal.Scopes) != 3 {
		t.Fatalf("unexpected scopes: %v", principal.Scopes)
	}
}

func TestOIDCAuthenticatorRejectsInvalidClaims(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mutate func(*accessTokenClaims)
	}{
		{name: "wrong issuer", mutate: func(claims *accessTokenClaims) { claims.Issuer = "https://wrong.invalid/" }},
		{name: "wrong audience", mutate: func(claims *accessTokenClaims) { claims.Audience = jwt.ClaimStrings{"wrong"} }},
		{name: "expired", mutate: func(claims *accessTokenClaims) {
			claims.ExpiresAt = jwt.NewNumericDate(time.Date(2026, time.August, 30, 10, 0, 0, 0, time.UTC))
		}},
		{name: "not yet valid", mutate: func(claims *accessTokenClaims) {
			claims.NotBefore = jwt.NewNumericDate(time.Date(2026, time.August, 30, 13, 0, 0, 0, time.UTC))
		}},
		{name: "issued in future", mutate: func(claims *accessTokenClaims) {
			claims.IssuedAt = jwt.NewNumericDate(time.Date(2026, time.August, 30, 13, 0, 0, 0, time.UTC))
		}},
		{name: "missing issued at", mutate: func(claims *accessTokenClaims) { claims.IssuedAt = nil }},
		{name: "missing expiry", mutate: func(claims *accessTokenClaims) { claims.ExpiresAt = nil }},
		{name: "missing subject", mutate: func(claims *accessTokenClaims) { claims.Subject = "" }},
		{name: "control in subject", mutate: func(claims *accessTokenClaims) { claims.Subject = "user\nother" }},
		{name: "wrong client", mutate: func(claims *accessTokenClaims) { claims.ClientID = "attacker-client" }},
		{name: "Auth0 default azp profile", mutate: func(claims *accessTokenClaims) { claims.AuthorizedFor = claims.ClientID; claims.ClientID = "" }},
		{name: "mixed client profiles", mutate: func(claims *accessTokenClaims) { claims.AuthorizedFor = claims.ClientID }},
		{name: "unverified email", mutate: func(claims *accessTokenClaims) { unverified := false; claims.EmailVerified = &unverified }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newOIDCFixture(t)
			key := testPrivateKey(t)
			fixture.addKey("current", key)
			if _, err := fixture.authenticator.Authenticate(context.Background(), signAccessToken(t, fixture, key, "current", testCase.mutate)); err == nil {
				t.Fatal("expected token rejection")
			}
		})
	}
}

func TestOIDCAuthenticatorRejectsWrongAlgorithm(t *testing.T) {
	fixture := newOIDCFixture(t)
	claims := jwt.MapClaims{
		"iss": fixture.issuer, "sub": "auth0|synthetic-user", "aud": "https://api.test.invalid",
		"exp": fixture.now.Add(time.Minute).Unix(), "iat": fixture.now.Unix(), "client_id": "mobile-client",
	}
	hmac := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	hmac.Header["kid"] = "current"
	hmac.Header["typ"] = "at+jwt"
	raw, err := hmac.SignedString([]byte("synthetic-test-key"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.authenticator.Authenticate(context.Background(), raw); err == nil {
		t.Fatal("expected HS256 rejection")
	}

}

func TestOIDCAuthenticatorRejectsUnconfiguredTokenTypes(t *testing.T) {
	fixture := newOIDCFixture(t)
	key := testPrivateKey(t)
	fixture.addKey("current", key)
	for name, tokenType := range map[string]any{
		"Auth0 default profile": "JWT",
		"ID token profile":      "id+jwt",
		"media type alias":      "application/at+jwt",
		"missing type":          nil,
	} {
		t.Run(name, func(t *testing.T) {
			claimsToken := jwt.NewWithClaims(jwt.SigningMethodRS256, &accessTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer: fixture.issuer, Subject: "auth0|synthetic-user",
					Audience:  jwt.ClaimStrings{"https://api.test.invalid"},
					ExpiresAt: jwt.NewNumericDate(fixture.now.Add(time.Minute)),
					IssuedAt:  jwt.NewNumericDate(fixture.now),
				},
				Scope: "profile:read", ClientID: "mobile-client",
			})
			claimsToken.Header["kid"] = "current"
			claimsToken.Header["typ"] = tokenType
			raw, err := claimsToken.SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := fixture.authenticator.Authenticate(context.Background(), raw); err == nil {
				t.Fatalf("expected token type %v rejection", tokenType)
			}
		})
	}
}

func TestOIDCAuthenticatorRefreshesOnceForRotatedAndUnknownKeys(t *testing.T) {
	fixture := newOIDCFixture(t)
	oldKey := testPrivateKey(t)
	newKey := testPrivateKey(t)
	unknownKey := testPrivateKey(t)
	fixture.addKey("old", oldKey)
	if _, err := fixture.authenticator.Authenticate(context.Background(), signAccessToken(t, fixture, oldKey, "old", nil)); err != nil {
		t.Fatal(err)
	}
	if fixture.requestCount() != 1 {
		t.Fatalf("initial JWKS requests = %d, want 1", fixture.requestCount())
	}
	fixture.replaceKey("new", newKey)
	if _, err := fixture.authenticator.Authenticate(context.Background(), signAccessToken(t, fixture, newKey, "new", nil)); err != nil {
		t.Fatalf("rotated key rejected: %v", err)
	}
	if fixture.requestCount() != 2 {
		t.Fatalf("JWKS requests after rotation = %d, want 2", fixture.requestCount())
	}
	if _, err := fixture.authenticator.Authenticate(context.Background(), signAccessToken(t, fixture, unknownKey, "unknown", nil)); err == nil {
		t.Fatal("expected unknown key rejection")
	}
	if fixture.requestCount() != 2 {
		t.Fatalf("unknown-key refresh was not bounded; requests = %d", fixture.requestCount())
	}
}

func TestOIDCAuthenticatorRefreshesAReusedKeyIDAfterSignatureFailure(t *testing.T) {
	fixture := newOIDCFixture(t)
	oldKey := testPrivateKey(t)
	newKey := testPrivateKey(t)
	fixture.addKey("stable", oldKey)
	if _, err := fixture.authenticator.Authenticate(context.Background(), signAccessToken(t, fixture, oldKey, "stable", nil)); err != nil {
		t.Fatal(err)
	}
	fixture.replaceKey("stable", newKey)
	if _, err := fixture.authenticator.Authenticate(context.Background(), signAccessToken(t, fixture, newKey, "stable", nil)); err != nil {
		t.Fatalf("reused kid rotation rejected: %v", err)
	}
	if fixture.requestCount() != 2 {
		t.Fatalf("JWKS requests=%d, want one bounded signature refresh", fixture.requestCount())
	}
}

func TestOIDCAuthenticatorBacksOffInitialDiscoveryFailures(t *testing.T) {
	fixture := newOIDCFixture(t)
	key := testPrivateKey(t)
	fixture.mu.Lock()
	fixture.failDiscovery = true
	fixture.mu.Unlock()
	raw := signAccessToken(t, fixture, key, "unavailable", nil)
	for range 2 {
		if _, err := fixture.authenticator.Authenticate(context.Background(), raw); err == nil {
			t.Fatal("expected discovery failure")
		}
	}
	if fixture.discoveryRequestCount() != 1 {
		t.Fatalf("discovery requests=%d, want one bounded attempt", fixture.discoveryRequestCount())
	}
}

func TestOIDCAuthenticatorRejectsCriticalHeadersAndOversizedTokens(t *testing.T) {
	fixture := newOIDCFixture(t)
	key := testPrivateKey(t)
	fixture.addKey("current", key)
	verified := true
	claims := accessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: fixture.issuer, Subject: "auth0|synthetic-user",
			Audience: jwt.ClaimStrings{"https://api.test.invalid"}, ExpiresAt: jwt.NewNumericDate(fixture.now.Add(time.Minute)),
			IssuedAt: jwt.NewNumericDate(fixture.now),
		},
		Scope: "profile:read", ClientID: "mobile-client", EmailVerified: &verified,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "current"
	token.Header["typ"] = "at+jwt"
	token.Header["crit"] = []string{"synthetic"}
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.authenticator.Authenticate(context.Background(), raw); err == nil {
		t.Fatal("expected critical JOSE header rejection")
	}
	if _, err := fixture.authenticator.Authenticate(context.Background(), strings.Repeat("x", maximumAccessTokenBytes+1)); err == nil {
		t.Fatal("expected oversized token rejection")
	}
}

func TestOIDCAuthenticatorRejectsNonCanonicalConfiguration(t *testing.T) {
	for name, options := range map[string]OIDCOptions{
		"issuer user info": {Issuer: "https://user@issuer.invalid/", Audience: "https://api.invalid", AllowedClientIDs: []string{"client"}},
		"issuer path":      {Issuer: "https://issuer.invalid/path/", Audience: "https://api.invalid", AllowedClientIDs: []string{"client"}},
		"trimmed audience": {Issuer: "https://issuer.invalid/", Audience: " https://api.invalid", AllowedClientIDs: []string{"client"}},
		"duplicate clients": {Issuer: "https://issuer.invalid/", Audience: "https://api.invalid",
			AllowedClientIDs: []string{"client", "client"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewOIDCAuthenticator(options); err == nil {
				t.Fatal("expected configuration rejection")
			}
		})
	}
}

func TestNormalizeScopesRejectsControlCharacters(t *testing.T) {
	if _, ok := normalizeScopes("profile:read\nprofile:write"); ok {
		t.Fatal("expected control-character scope rejection")
	}
	if _, ok := normalizeScopes("profile:read\\other"); ok {
		t.Fatal("expected invalid RFC 6749 scope-token rejection")
	}
}
