package tplinkomadasdkforgo

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	kiotaauth "github.com/microsoft/kiota-abstractions-go/authentication"
	khttp "github.com/microsoft/kiota-http-go"

	"github.com/NerdIT-Tech/tplink-omada-sdk-for-go/auth"
)

// NewInsecureHTTPClient returns an *http.Client that skips TLS certificate
// verification. Omada controllers are typically self-hosted on a local network with
// a self-signed certificate, so this is a convenience for that common case; prefer a
// properly configured *http.Client (e.g. with the controller's CA trusted) whenever
// possible.
func NewInsecureHTTPClient() *http.Client {
	client := khttp.GetDefaultClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport == nil {
		transport = &http.Transport{}
	} else {
		transport = transport.Clone()
	}
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	transport.TLSClientConfig.InsecureSkipVerify = true
	client.Transport = transport
	return client
}

// NewWithAccessToken creates an OmadaApiClient that authenticates every request with
// a caller-supplied bearer access token, e.g. one an application obtained and manages
// itself. baseURL is the controller's Open API base URL (e.g.
// "https://192.168.1.1:8043"). If httpClient is nil, a default client with TLS
// verification enabled is used; pass NewInsecureHTTPClient() for controllers with
// self-signed certificates.
func NewWithAccessToken(baseURL, accessToken string, httpClient *http.Client) (*OmadaApiClient, error) {
	if httpClient == nil {
		httpClient = khttp.GetDefaultClient()
	}
	provider := auth.NewStaticTokenProvider(accessToken)
	return newClient(baseURL, auth.NewAuthenticationProvider(provider), httpClient)
}

// NewWithClientCredentials creates an OmadaApiClient that authenticates using an
// Omada Open API client ID and client secret (registered under Settings > Platform
// Integration on the controller). The access token is fetched on first use and
// transparently refreshed before it expires. omadacID scopes the credentials to a
// specific Omada Cloud Controller ID; pass "" if not applicable. If httpClient is
// nil, a default client with TLS verification enabled is used; pass
// NewInsecureHTTPClient() for controllers with self-signed certificates.
func NewWithClientCredentials(baseURL, clientID, clientSecret, omadacID string, httpClient *http.Client) (*OmadaApiClient, error) {
	if httpClient == nil {
		httpClient = khttp.GetDefaultClient()
	}
	provider := auth.NewClientCredentialsTokenProvider(baseURL, clientID, clientSecret, omadacID, httpClient)
	return newClient(baseURL, auth.NewAuthenticationProvider(provider), httpClient)
}

func newClient(baseURL string, authProvider kiotaauth.AuthenticationProvider, httpClient *http.Client) (*OmadaApiClient, error) {
	httpClient = withOmadaResponseNormalization(httpClient)
	adapter, err := khttp.NewNetHttpRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authProvider, nil, nil, httpClient)
	if err != nil {
		return nil, fmt.Errorf("omada sdk: creating request adapter: %w", err)
	}
	if baseURL != "" {
		adapter.SetBaseUrl(baseURL)
	}
	return NewOmadaApiClient(adapter), nil
}

// omadaResponseNormalizingTransport corrects two observed inconsistencies in the
// Omada Open API's HTTP responses that otherwise cause the generated Kiota request
// adapter to silently lose information:
//
//  1. Missing Content-Type. Most responses — success or controller-level error alike
//     — are labeled "application/json". But at least one error path (an unrecognized
//     omadacId) sends a well-formed JSON envelope with no Content-Type header at all.
//     Kiota picks its JSON parser by Content-Type; with none, it can't identify the
//     body and returns (nil, nil) for the whole call — no error, no data — even
//     though the response was a perfectly good envelope. A caller that doesn't
//     defensively nil-check the result nil-pointer-panics.
//  2. Envelope wrapped in a non-2xx status. Most controller-level errors (an invalid
//     omadacId, an out-of-range page, an expired token) arrive as the controller's
//     own {"errorCode":...,"msg":...} envelope wrapped in HTTP 200, and the generated
//     builders deserialize that normally. But for some failures — observed with a
//     missing Authorization header — the controller sends that same envelope shape
//     wrapped in a non-2xx status (e.g. 401) instead. Because no error mapping is
//     registered for those statuses, the adapter discards the response body entirely
//     and returns a generic "unexpected status code" error, hiding the controller's
//     actual errorCode and msg.
//
// This transport reads the (typically small, paginated) response body once and: sets
// Content-Type to application/json when it's missing but the body looks like JSON,
// and rewrites the status to 200 when the body is a recognizable Omada envelope. Both
// fixes make errors reach the caller through the normal OperationResponse result, the
// same way every other controller error already does — rather than as a missing or
// generic error.
type omadaResponseNormalizingTransport struct {
	base http.RoundTripper
}

func (t *omadaResponseNormalizingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp == nil || resp.Body == nil {
		return resp, err
	}

	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		resp.Body = io.NopCloser(bytes.NewReader(nil))
		return resp, nil
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))

	if resp.Header.Get("Content-Type") == "" && looksLikeJSON(body) {
		resp.Header.Set("Content-Type", "application/json")
	}

	if resp.StatusCode >= 400 && isOmadaErrorEnvelope(body) {
		resp.StatusCode = http.StatusOK
		resp.Status = "200 OK"
	}
	return resp, nil
}

// looksLikeJSON reports whether body appears to be a JSON object or array, ignoring
// leading whitespace.
func looksLikeJSON(body []byte) bool {
	trimmed := bytes.TrimLeft(body, " \t\r\n")
	return len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
}

// isOmadaErrorEnvelope reports whether body is a well-formed Omada Open API response
// envelope — it has an integer "errorCode" field, the one property every such
// envelope carries — as opposed to some other error page (e.g. an API gateway's
// generic "Bad Request" body for a missing required query parameter) that the SDK has
// no typed model for and so cannot usefully rewrite.
func isOmadaErrorEnvelope(body []byte) bool {
	var probe struct {
		ErrorCode *int32 `json:"errorCode"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return false
	}
	return probe.ErrorCode != nil
}

// withOmadaResponseNormalization returns an *http.Client equivalent to client but
// with omadaResponseNormalizingTransport applied on top of its existing transport.
func withOmadaResponseNormalization(client *http.Client) *http.Client {
	wrapped := *client
	base := wrapped.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	wrapped.Transport = &omadaResponseNormalizingTransport{base: base}
	return &wrapped
}
