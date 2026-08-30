package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultOIDCCacheTTL       = 15 * time.Minute
	defaultOIDCRequestTimeout = 5 * time.Second
	minimumForcedRefreshGap   = 5 * time.Second
	maximumOIDCResponseBytes  = 1 << 20
	maximumAccessTokenBytes   = 16 << 10
)

type OIDCOptions struct {
	Issuer                    string
	Audience                  string
	AllowedClientIDs          []string
	HTTPClient                *http.Client
	CacheTTL                  time.Duration
	ClockSkew                 time.Duration
	Now                       func() time.Time
	AllowInsecureIssuerInTest bool
}

type OIDCAuthenticator struct {
	issuer           string
	audience         string
	allowedClients   map[string]struct{}
	client           *http.Client
	cacheTTL         time.Duration
	clockSkew        time.Duration
	now              func() time.Time
	allowInsecure    bool
	issuerURL        *url.URL
	mu               sync.Mutex
	jwksURI          string
	keys             map[string]*rsa.PublicKey
	keysExpireAt     time.Time
	lastForcedReload time.Time
	lastLoadFailure  time.Time
}

type accessTokenClaims struct {
	jwt.RegisteredClaims
	Scope         string `json:"scope"`
	ClientID      string `json:"client_id"`
	AuthorizedFor string `json:"azp"`
	EmailVerified *bool  `json:"email_verified"`
}

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

type jwksDocument struct {
	Keys []jsonWebKey `json:"keys"`
}

type jsonWebKey struct {
	KeyType   string `json:"kty"`
	KeyID     string `json:"kid"`
	Use       string `json:"use"`
	Algorithm string `json:"alg"`
	Modulus   string `json:"n"`
	Exponent  string `json:"e"`
}

func NewOIDCAuthenticator(options OIDCOptions) (*OIDCAuthenticator, error) {
	issuerURL, err := url.Parse(options.Issuer)
	if err != nil || len(options.Issuer) > 80 || issuerURL.Scheme == "" || issuerURL.Host == "" || issuerURL.User != nil || issuerURL.Path != "/" || issuerURL.RawQuery != "" || issuerURL.Fragment != "" {
		return nil, errors.New("OIDC issuer must be an absolute URL without query or fragment")
	}
	if options.Issuer != issuerURL.String() || !strings.HasSuffix(options.Issuer, "/") {
		return nil, errors.New("OIDC issuer must be canonical and end with a slash")
	}
	if issuerURL.Scheme != "https" && !options.AllowInsecureIssuerInTest {
		return nil, errors.New("OIDC issuer must use HTTPS")
	}
	if options.Audience == "" || strings.TrimSpace(options.Audience) != options.Audience || len(options.Audience) > 255 {
		return nil, errors.New("OIDC audience is required")
	}
	allowedClients := make(map[string]struct{}, len(options.AllowedClientIDs))
	for _, clientID := range options.AllowedClientIDs {
		if clientID == "" || strings.TrimSpace(clientID) != clientID || len(clientID) > 255 || hasControl(clientID) {
			return nil, errors.New("OIDC allowed client IDs must be non-empty and at most 255 characters")
		}
		if _, duplicate := allowedClients[clientID]; duplicate {
			return nil, errors.New("OIDC allowed client IDs must be distinct")
		}
		allowedClients[clientID] = struct{}{}
	}
	if len(allowedClients) == 0 {
		return nil, errors.New("at least one OIDC client ID is required")
	}
	client := &http.Client{}
	if options.HTTPClient != nil {
		*client = *options.HTTPClient
	}
	if client.Timeout <= 0 || client.Timeout > defaultOIDCRequestTimeout {
		client.Timeout = defaultOIDCRequestTimeout
	}
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return errors.New("OIDC redirects are not allowed")
	}
	cacheTTL := options.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultOIDCCacheTTL
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &OIDCAuthenticator{
		issuer: options.Issuer, audience: options.Audience, allowedClients: allowedClients,
		client: client, cacheTTL: cacheTTL, clockSkew: options.ClockSkew, now: now,
		allowInsecure: options.AllowInsecureIssuerInTest, issuerURL: issuerURL,
	}, nil
}

func (a *OIDCAuthenticator) Authenticate(ctx context.Context, raw string) (Principal, error) {
	if len(raw) == 0 || len(raw) > maximumAccessTokenBytes || strings.TrimSpace(raw) != raw {
		return Principal{}, ErrUnauthenticated
	}
	claims, token, err := a.parseAccessToken(ctx, raw)
	if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
		if _, refreshErr := a.loadKeys(ctx, true); refreshErr == nil {
			claims, token, err = a.parseAccessToken(ctx, raw)
		}
	}
	if err != nil || token == nil || !token.Valid {
		return Principal{}, ErrUnauthenticated
	}
	if claims.Subject == "" || len(claims.Subject) > 255 || strings.TrimSpace(claims.Subject) != claims.Subject || hasControl(claims.Subject) || claims.IssuedAt == nil {
		return Principal{}, ErrUnauthenticated
	}
	if claims.EmailVerified != nil && !*claims.EmailVerified {
		return Principal{}, ErrEmailUnverified
	}
	clientID, ok := a.validClientID(claims.ClientID, claims.AuthorizedFor)
	if !ok {
		return Principal{}, ErrUnauthenticated
	}
	scopes, ok := normalizeScopes(claims.Scope)
	if !ok {
		return Principal{}, ErrUnauthenticated
	}
	return Principal{
		Provider: a.issuer, Subject: claims.Subject, ClientID: clientID,
		Scopes: scopes, Roles: []string{"user"},
	}, nil
}

func (a *OIDCAuthenticator) parseAccessToken(ctx context.Context, raw string) (*accessTokenClaims, *jwt.Token, error) {
	claims := new(accessTokenClaims)
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(a.issuer),
		jwt.WithAudience(a.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(a.clockSkew),
		jwt.WithTimeFunc(a.now),
	)
	token, err := parser.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 || token.Header["alg"] != jwt.SigningMethodRS256.Alg() {
			return nil, ErrUnauthenticated
		}
		if _, critical := token.Header["crit"]; critical {
			return nil, ErrUnauthenticated
		}
		typ, ok := token.Header["typ"].(string)
		if !ok || typ != "at+jwt" {
			return nil, ErrUnauthenticated
		}
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" || len(kid) > 128 || strings.TrimSpace(kid) != kid || hasControl(kid) {
			return nil, ErrUnauthenticated
		}
		return a.key(ctx, kid)
	})
	return claims, token, err
}

func (a *OIDCAuthenticator) validClientID(clientID, authorizedParty string) (string, bool) {
	if clientID == "" || authorizedParty != "" {
		return "", false
	}
	_, ok := a.allowedClients[clientID]
	return clientID, ok
}

func normalizeScopes(raw string) ([]string, bool) {
	if raw == "" {
		return []string{}, true
	}
	fields := strings.Split(raw, " ")
	seen := make(map[string]struct{}, len(fields))
	result := make([]string, 0, len(fields))
	for _, scope := range fields {
		if len(scope) > 128 || !validScopeToken(scope) {
			return nil, false
		}
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		result = append(result, scope)
	}
	return result, true
}

func validScopeToken(scope string) bool {
	if scope == "" {
		return false
	}
	for index := range len(scope) {
		value := scope[index]
		if value < 0x21 || value > 0x7e || value == 0x22 || value == 0x5c {
			return false
		}
	}
	return true
}

func hasControl(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return true
		}
	}
	return false
}

func (a *OIDCAuthenticator) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	keys, err := a.loadKeys(ctx, false)
	if err != nil {
		return nil, ErrUnauthenticated
	}
	if key := keys[kid]; key != nil {
		return key, nil
	}
	keys, refreshErr := a.loadKeys(ctx, true)
	if refreshErr != nil {
		return nil, ErrUnauthenticated
	}
	if key := keys[kid]; key != nil {
		return key, nil
	}
	return nil, ErrUnauthenticated
}

func (a *OIDCAuthenticator) loadKeys(ctx context.Context, force bool) (map[string]*rsa.PublicKey, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	if len(a.keys) == 0 && !a.lastLoadFailure.IsZero() && now.Before(a.lastLoadFailure.Add(minimumForcedRefreshGap)) {
		return nil, errors.New("OIDC key loading is temporarily unavailable")
	}
	if !force && len(a.keys) > 0 && now.Before(a.keysExpireAt) {
		return a.keys, nil
	}
	if force && len(a.keys) > 0 && now.Sub(a.lastForcedReload) < minimumForcedRefreshGap {
		return a.keys, nil
	}
	if force {
		a.lastForcedReload = now
	}
	if a.jwksURI == "" {
		uri, err := a.discover(ctx)
		if err != nil {
			a.lastLoadFailure = now
			return nil, err
		}
		a.jwksURI = uri
	}
	keys, err := a.fetchKeys(ctx, a.jwksURI)
	if err != nil {
		a.lastLoadFailure = now
		return nil, err
	}
	a.keys = keys
	a.lastLoadFailure = time.Time{}
	a.keysExpireAt = now.Add(a.cacheTTL)
	return a.keys, nil
}

func (a *OIDCAuthenticator) discover(ctx context.Context) (string, error) {
	discoveryURL := a.issuer + ".well-known/openid-configuration"
	var document discoveryDocument
	if err := a.getJSON(ctx, discoveryURL, &document); err != nil {
		return "", fmt.Errorf("fetch OIDC discovery: %w", err)
	}
	if document.Issuer != a.issuer {
		return "", errors.New("OIDC discovery issuer mismatch")
	}
	jwksURL, err := url.Parse(document.JWKSURI)
	if err != nil || jwksURL.Scheme == "" || jwksURL.Host == "" || jwksURL.User != nil || jwksURL.RawQuery != "" || jwksURL.Fragment != "" || jwksURL.String() != document.JWKSURI {
		return "", errors.New("OIDC discovery returned an invalid JWKS URI")
	}
	if jwksURL.Scheme != a.issuerURL.Scheme || !strings.EqualFold(jwksURL.Host, a.issuerURL.Host) {
		return "", errors.New("OIDC JWKS URI must share the issuer origin")
	}
	if jwksURL.Scheme != "https" && !a.allowInsecure {
		return "", errors.New("OIDC JWKS URI must use HTTPS")
	}
	return jwksURL.String(), nil
}

func (a *OIDCAuthenticator) fetchKeys(ctx context.Context, uri string) (map[string]*rsa.PublicKey, error) {
	var document jwksDocument
	if err := a.getJSON(ctx, uri, &document); err != nil {
		return nil, fmt.Errorf("fetch OIDC JWKS: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, encoded := range document.Keys {
		if encoded.KeyType != "RSA" || encoded.KeyID == "" || len(encoded.KeyID) > 128 || strings.TrimSpace(encoded.KeyID) != encoded.KeyID || hasControl(encoded.KeyID) {
			continue
		}
		if encoded.Use != "sig" {
			continue
		}
		if encoded.Algorithm != "RS256" {
			continue
		}
		if _, duplicate := keys[encoded.KeyID]; duplicate {
			return nil, errors.New("OIDC JWKS contains a duplicate key ID")
		}
		key, err := parseRSAKey(encoded.Modulus, encoded.Exponent)
		if err != nil {
			continue
		}
		keys[encoded.KeyID] = key
	}
	if len(keys) == 0 {
		return nil, errors.New("OIDC JWKS contains no usable RS256 keys")
	}
	return keys, nil
}

func parseRSAKey(modulus, exponent string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil || len(nBytes) < 256 || len(nBytes) > 1024 {
		return nil, errors.New("invalid RSA modulus")
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil || len(eBytes) == 0 || len(eBytes) > 4 {
		return nil, errors.New("invalid RSA exponent")
	}
	padded := make([]byte, 4)
	copy(padded[4-len(eBytes):], eBytes)
	e := int(binary.BigEndian.Uint32(padded))
	key := &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}
	if key.N.BitLen() < 2048 || key.E < 3 || key.E%2 == 0 {
		return nil, errors.New("weak RSA public key")
	}
	return key, nil
}

func (a *OIDCAuthenticator) getJSON(ctx context.Context, rawURL string, destination any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	response, err := a.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %d", response.StatusCode)
	}
	if response.ContentLength > maximumOIDCResponseBytes {
		return errors.New("OIDC response is too large")
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || (mediaType != "application/json" && mediaType != "application/jwk-set+json") {
		return errors.New("OIDC response must be JSON")
	}
	limited := &io.LimitedReader{R: response.Body, N: maximumOIDCResponseBytes + 1}
	decoder := json.NewDecoder(limited)
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("OIDC response contains trailing JSON")
	}
	if limited.N <= 0 {
		return errors.New("OIDC response is too large")
	}
	return nil
}
