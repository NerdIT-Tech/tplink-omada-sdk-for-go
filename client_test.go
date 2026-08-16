package tplinkomadasdkforgo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NerdIT-Tech/tplink-omada-sdk-for-go/auth"
	"github.com/NerdIT-Tech/tplink-omada-sdk-for-go/openapi"
)

func getSites(t *testing.T, baseURL, accessToken string) (int32, string, error) {
	t.Helper()
	client, err := NewWithAccessToken(baseURL, accessToken, nil)
	if err != nil {
		t.Fatalf("NewWithAccessToken: %v", err)
	}
	page, pageSize := int32(1), int32(10)
	resp, err := client.Openapi().V1().ByOmadacId("does-not-matter").Sites().Get(context.Background(), &openapi.V1ItemSitesRequestBuilderGetRequestConfiguration{
		QueryParameters: &openapi.V1ItemSitesRequestBuilderGetQueryParameters{Page: &page, PageSize: &pageSize},
	})
	if err != nil || resp == nil {
		return 0, "", err
	}
	code, msg := int32(0), ""
	if resp.GetErrorCode() != nil {
		code = *resp.GetErrorCode()
	}
	if resp.GetMsg() != nil {
		msg = *resp.GetMsg()
	}
	return code, msg, nil
}

// TestErrorEnvelopeNormalization_NonStandardStatus reproduces what a live Omada
// controller does when a request reaches it with no Authorization header: it reports
// its own errorCode/msg envelope, the exact shape every successful call uses, but
// wraps it in an HTTP 401 instead of the 200 it uses for every other envelope-level
// error. Without normalization the caller only sees a generic, unhelpful
// "unexpected status code" error with none of that detail.
func TestErrorEnvelopeNormalization_NonStandardStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"msg":"Open API Authorized failed, please check whether the input parameters are legal.","errorCode":-44116}`))
	}))
	defer srv.Close()

	code, msg, err := getSites(t, srv.URL, "some-token")
	if err != nil {
		t.Fatalf("expected the envelope to surface through the normal result, got error: %v", err)
	}
	if code != -44116 {
		t.Errorf("expected errorCode -44116, got %d", code)
	}
	if msg == "" {
		t.Errorf("expected the controller's msg to survive, got empty string")
	}
}

// TestErrorEnvelopeNormalization_NonEnvelopeStatus confirms the transport leaves
// alone a non-2xx response that isn't a recognizable Omada envelope (e.g. a gateway's
// generic error page for a malformed request) — it must not be rewritten to a
// fabricated success.
func TestErrorEnvelopeNormalization_NonEnvelopeStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"timestamp":1,"status":400,"error":"Bad Request","path":"/openapi/v1/x/sites"}`))
	}))
	defer srv.Close()

	_, _, err := getSites(t, srv.URL, "some-token")
	if err == nil {
		t.Fatalf("expected a non-envelope 400 response to still surface as an error")
	}
}

// TestErrorEnvelopeNormalization_SuccessPassthrough confirms normal 200 responses are
// unaffected.
func TestErrorEnvelopeNormalization_SuccessPassthrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errorCode":0,"msg":"Success.","result":{"totalRows":0,"currentPage":1,"currentSize":10,"data":[]}}`))
	}))
	defer srv.Close()

	code, msg, err := getSites(t, srv.URL, "some-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 || msg != "Success." {
		t.Errorf("expected passthrough of errorCode=0 msg=Success., got errorCode=%d msg=%q", code, msg)
	}
}

// TestResponseNormalization_MissingContentType reproduces what a live Omada
// controller does on at least one error path (an unrecognized omadacId): it sends a
// well-formed errorCode/msg envelope but omits the Content-Type header entirely.
// Without normalization, Kiota's request adapter can't identify the body as JSON and
// silently returns (nil, nil) for the whole call, discarding the envelope with no
// error at all.
func TestResponseNormalization_MissingContentType(t *testing.T) {
	// net/http's ResponseWriter auto-sniffs and sets a Content-Type when a handler
	// never sets one, which would mask the bug this test targets. Hijack the
	// connection to write a truly header-bare raw response, matching what the live
	// controller was observed to send.
	body := `{"msg":"Controller ID not exist.","errorCode":-7131}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, buf, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		defer conn.Close()
		fmt.Fprintf(buf, "HTTP/1.1 200 OK\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(body), body)
		buf.Flush()
	}))
	defer srv.Close()

	code, msg, err := getSites(t, srv.URL, "some-token")
	if err != nil {
		t.Fatalf("expected the envelope to surface through the normal result, got error: %v", err)
	}
	if code != -7131 {
		t.Errorf("expected errorCode -7131, got %d", code)
	}
	if msg == "" {
		t.Errorf("expected the controller's msg to survive, got empty string")
	}
}

// TestEmptyAccessTokenFailsFast confirms a client configured with an empty access
// token fails locally with auth.ErrEmptyToken instead of silently sending the
// controller an unauthenticated request.
func TestEmptyAccessTokenFailsFast(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, _, err := getSites(t, srv.URL, "")
	if !errors.Is(err, auth.ErrEmptyToken) {
		t.Fatalf("expected auth.ErrEmptyToken, got: %v", err)
	}
	if called {
		t.Errorf("expected no request to reach the server for an empty access token")
	}
}
