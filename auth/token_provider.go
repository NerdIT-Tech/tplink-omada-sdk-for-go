// Package auth provides authentication.AccessTokenProvider implementations for the
// Omada Open API, which secures requests with a bearer access token obtained either
// directly from application code or via an OAuth2-style client credentials exchange
// against the controller's /openapi/authorize/token endpoint.
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
)

// expiryLeeway is subtracted from a token's reported lifetime so a refresh happens
// slightly before the controller actually rejects the old token.
const expiryLeeway = 60 * time.Second

// StaticTokenProvider serves a fixed, caller-managed bearer token. Use it when the
// application already owns an Omada access token (and its refresh lifecycle) and
// simply needs it attached to outgoing requests.
type StaticTokenProvider struct {
	mu        sync.RWMutex
	token     string
	validator absauth.AllowedHostsValidator
}

var _ absauth.AccessTokenProvider = (*StaticTokenProvider)(nil)

// NewStaticTokenProvider creates a provider that authenticates every request with
// accessToken. allowedHosts restricts which hosts the token may be sent to; pass no
// hosts to allow any host.
func NewStaticTokenProvider(accessToken string, allowedHosts ...string) *StaticTokenProvider {
	return &StaticTokenProvider{
		token:     accessToken,
		validator: absauth.NewAllowedHostsValidator(allowedHosts),
	}
}

// SetToken updates the token used for subsequent requests, e.g. after the
// application refreshes it out of band.
func (p *StaticTokenProvider) SetToken(accessToken string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.token = accessToken
}

// GetAuthorizationToken returns the current access token.
func (p *StaticTokenProvider) GetAuthorizationToken(_ context.Context, _ *url.URL, _ map[string]interface{}) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.token, nil
}

// GetAllowedHostsValidator returns the hosts validator.
func (p *StaticTokenProvider) GetAllowedHostsValidator() *absauth.AllowedHostsValidator {
	return &p.validator
}

// tokenResult mirrors the "result" object returned by the Omada
// /openapi/authorize/token endpoint.
type tokenResult struct {
	AccessToken  string `json:"accessToken"`
	TokenType    string `json:"tokenType"`
	ExpiresIn    int64  `json:"expiresIn"`
	RefreshToken string `json:"refreshToken"`
}

// tokenEnvelope mirrors the standard Omada Open API response wrapper.
type tokenEnvelope struct {
	ErrorCode int32       `json:"errorCode"`
	Msg       string      `json:"msg"`
	Result    tokenResult `json:"result"`
}

// ClientCredentialsTokenProvider authenticates using an Omada Open API client ID and
// client secret (registered under Settings > Platform Integration on the
// controller). It exchanges those credentials for a bearer access token on first use
// and transparently refreshes it before it expires.
type ClientCredentialsTokenProvider struct {
	baseURL      string
	clientID     string
	clientSecret string
	omadacID     string
	httpClient   *http.Client
	validator    absauth.AllowedHostsValidator

	mu           sync.Mutex
	accessToken  string
	refreshToken string
	expiresAt    time.Time
}

var _ absauth.AccessTokenProvider = (*ClientCredentialsTokenProvider)(nil)

// NewClientCredentialsTokenProvider creates a provider that fetches and refreshes
// access tokens from the Omada controller at baseURL (e.g.
// "https://192.168.1.1:8043") using the given Open API client ID/secret. omadacID is
// the Omada Cloud Controller ID the credentials are scoped to; pass "" if the
// credentials are not scoped to a specific controller. If httpClient is nil,
// http.DefaultClient is used for the token exchange itself.
func NewClientCredentialsTokenProvider(baseURL, clientID, clientSecret, omadacID string, httpClient *http.Client) *ClientCredentialsTokenProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	allowedHosts := []string{}
	if u, err := url.Parse(baseURL); err == nil && u.Host != "" {
		allowedHosts = append(allowedHosts, u.Host)
	}
	return &ClientCredentialsTokenProvider{
		baseURL:      strings.TrimRight(baseURL, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		omadacID:     omadacID,
		httpClient:   httpClient,
		validator:    absauth.NewAllowedHostsValidator(allowedHosts),
	}
}

// GetAuthorizationToken returns a valid access token, fetching or refreshing it as
// needed.
func (p *ClientCredentialsTokenProvider) GetAuthorizationToken(ctx context.Context, _ *url.URL, _ map[string]interface{}) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.accessToken != "" && time.Now().Before(p.expiresAt) {
		return p.accessToken, nil
	}

	// The controller's refresh_token grant is undocumented and not reliably
	// exercisable, so tokens are simply re-issued via client_credentials once
	// they are close to expiry.
	if err := p.authorize(ctx); err != nil {
		return "", err
	}
	return p.accessToken, nil
}

// GetAllowedHostsValidator returns the hosts validator.
func (p *ClientCredentialsTokenProvider) GetAllowedHostsValidator() *absauth.AllowedHostsValidator {
	return &p.validator
}

// authorizeRequestBody is the JSON body the /openapi/authorize/token endpoint
// expects; the grant type itself is passed as a query parameter.
type authorizeRequestBody struct {
	OmadacID     string `json:"omadacId,omitempty"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func (p *ClientCredentialsTokenProvider) authorize(ctx context.Context) error {
	body, err := json.Marshal(authorizeRequestBody{ // #nosec G117 -- ClientSecret is the runtime-provided OAuth client secret being marshaled into the token request body, not a hardcoded credential
		OmadacID:     p.omadacID,
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
	})
	if err != nil {
		return fmt.Errorf("omada auth: encoding token request: %w", err)
	}
	return p.requestToken(ctx, "client_credentials", body)
}

func (p *ClientCredentialsTokenProvider) requestToken(ctx context.Context, grantType string, body []byte) error {
	endpoint := p.baseURL + "/openapi/authorize/token?" + (url.Values{"grant_type": {grantType}}).Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("omada auth: building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("omada auth: requesting token: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("omada auth: reading token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("omada auth: token request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var envelope tokenEnvelope
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("omada auth: decoding token response: %w", err)
	}
	if envelope.ErrorCode != 0 {
		return fmt.Errorf("omada auth: controller rejected token request (errorCode %d): %s", envelope.ErrorCode, envelope.Msg)
	}
	if envelope.Result.AccessToken == "" {
		return fmt.Errorf("omada auth: token response did not contain an access token")
	}

	p.accessToken = envelope.Result.AccessToken
	p.refreshToken = envelope.Result.RefreshToken
	lifetime := time.Duration(envelope.Result.ExpiresIn) * time.Second
	if lifetime > expiryLeeway {
		lifetime -= expiryLeeway
	}
	p.expiresAt = time.Now().Add(lifetime)
	return nil
}
