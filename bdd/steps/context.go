package steps

import (
	"crypto/tls"
	"net/http"
	"strings"
	"sync"

	sdk "github.com/NerdIT-Tech/tplink-omada-sdk-for-go"
	"github.com/NerdIT-Tech/tplink-omada-sdk-for-go/models"
)

// envelope is the shape common to every Omada Open API response: an errorCode/msg
// pair, wrapping a type-specific result. Steps that only care about success/failure
// can operate on this interface regardless of which endpoint produced the response.
type envelope interface {
	GetErrorCode() *int32
	GetMsg() *string
}

// capturingTransport wraps an http.RoundTripper and remembers every request it saw,
// so a step can assert on what was actually sent over the wire (e.g. that a bearer
// token was attached, or how many times a particular endpoint was hit).
type capturingTransport struct {
	base     http.RoundTripper
	mu       sync.Mutex
	requests []*http.Request
}

func (t *capturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.requests = append(t.requests, req.Clone(req.Context()))
	t.mu.Unlock()
	return t.base.RoundTrip(req)
}

func (t *capturingTransport) LastAuthorizationHeader() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.requests) == 0 {
		return ""
	}
	return t.requests[len(t.requests)-1].Header.Get("Authorization")
}

// RequestCount returns the total number of requests observed.
func (t *capturingTransport) RequestCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.requests)
}

// CountRequestsToPath returns how many observed requests had a URL path containing
// pathSubstring (e.g. "/authorize/token").
func (t *capturingTransport) CountRequestsToPath(pathSubstring string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	count := 0
	for _, req := range t.requests {
		if strings.Contains(req.URL.Path, pathSubstring) {
			count++
		}
	}
	return count
}

// omadaWorld carries state across the steps of a single scenario.
type omadaWorld struct {
	env *testEnv

	httpClient *http.Client
	capture    *capturingTransport

	clientID     string
	clientSecret string

	sdkClient *sdk.OmadaApiClient

	siteID string

	lastErr          error
	lastToken        string
	lastEnvelope     envelope
	sitesResponse    models.OperationResponseGridVOSiteSummaryInfoable
	devicesResponse  models.OperationResponseGridVODeviceInfoable
	concurrentTokens []string
}

func newOmadaWorld() (*omadaWorld, error) {
	env, err := loadTestEnv()
	if err != nil {
		return nil, err
	}
	return &omadaWorld{env: env, clientID: env.ClientID, clientSecret: env.ClientSecret}, nil
}

// insecureHTTPClient mirrors sdk.NewInsecureHTTPClient but keeps a capturing
// transport in the chain so steps can inspect outgoing requests.
func (w *omadaWorld) insecureHTTPClient() *http.Client {
	base := sdk.NewInsecureHTTPClient()
	w.capture = &capturingTransport{base: base.Transport}
	base.Transport = w.capture
	return base
}

func secureVerifyingHTTPClient() *http.Client {
	return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{}}}
}
