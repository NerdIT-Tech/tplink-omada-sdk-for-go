// Package steps implements godog step definitions that exercise the generated
// Omada SDK against a real, reachable Omada controller. The controller's address
// and Open API client credentials are read from the repo-root .env file.
package steps

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	sdk "github.com/NerdIT-Tech/tplink-omada-sdk-for-go"
	"github.com/NerdIT-Tech/tplink-omada-sdk-for-go/auth"
	"github.com/NerdIT-Tech/tplink-omada-sdk-for-go/openapi"

	"github.com/cucumber/godog"
)

// InitializeScenario registers every step definition used by the .feature files
// under bdd/features against sc, wiring them to a shared *omadaWorld per scenario.
func InitializeScenario(sc *godog.ScenarioContext) {
	var w *omadaWorld

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		world, err := newOmadaWorld()
		if err != nil {
			return ctx, err
		}
		w = world
		return ctx, nil
	})

	// --- authentication.feature -------------------------------------------------

	sc.Step(`^an Omada controller reachable at the configured base URL$`, func() error {
		return nil // loadTestEnv already validated BASE_URL is set.
	})

	sc.Step(`^the controller's Omada Cloud Controller ID$`, func(ctx context.Context) error {
		id, err := discoverOmadacID(ctx, w.insecureHTTPClient(), w.env.BaseURL)
		if err != nil {
			return fmt.Errorf("discovering omadacId: %w", err)
		}
		w.env.OmadacID = id
		return nil
	})

	sc.Step(`^a client configured with the controller's client ID and client secret$`, func() error {
		w.clientID = w.env.ClientID
		w.clientSecret = w.env.ClientSecret
		return nil
	})

	sc.Step(`^a client configured with the controller's client ID and an invalid client secret$`, func() error {
		w.clientID = w.env.ClientID
		w.clientSecret = w.env.ClientSecret + "-deliberately-invalid"
		return nil
	})

	sc.Step(`^a client configured with an unknown client ID and the controller's client secret$`, func() error {
		w.clientID = w.env.ClientID + "-deliberately-unknown"
		w.clientSecret = w.env.ClientSecret
		return nil
	})

	sc.Step(`^a client configured with an empty access token$`, func(_ context.Context) error {
		client, err := sdk.NewWithAccessToken(w.env.BaseURL, "", w.insecureHTTPClient())
		if err != nil {
			return err
		}
		w.sdkClient = client
		return nil
	})

	sc.Step(`^the client authenticates with the controller$`, func(ctx context.Context) error {
		provider := auth.NewClientCredentialsTokenProvider(w.env.BaseURL, w.clientID, w.clientSecret, w.env.OmadacID, w.insecureHTTPClient())
		token, err := provider.GetAuthorizationToken(ctx, nil, nil)
		w.lastToken = token
		w.lastErr = err
		return nil
	})

	sc.Step(`^the controller issues an access token$`, func() error {
		if w.lastErr != nil {
			return fmt.Errorf("expected an access token, got error: %w", w.lastErr)
		}
		if w.lastToken == "" {
			return errors.New("expected a non-empty access token")
		}
		return nil
	})

	sc.Step(`^the SDK attaches that access token to the next request using Omada's access token header scheme$`, func(ctx context.Context) error {
		client, err := sdk.NewWithAccessToken(w.env.BaseURL, w.lastToken, w.insecureHTTPClient())
		if err != nil {
			return err
		}
		page, pageSize := int32(1), int32(1)
		_, _ = client.Openapi().V1().ByOmadacId(w.env.OmadacID).Sites().Get(ctx, &openapi.V1ItemSitesRequestBuilderGetRequestConfiguration{
			QueryParameters: &openapi.V1ItemSitesRequestBuilderGetQueryParameters{Page: &page, PageSize: &pageSize},
		})
		got := w.capture.LastAuthorizationHeader()
		want := "AccessToken=" + w.lastToken
		if got != want {
			return fmt.Errorf("expected Authorization header %q, got %q", want, got)
		}
		return nil
	})

	sc.Step(`^the controller reports an authentication error$`, func() error {
		if w.lastErr == nil {
			return errors.New("expected authentication to fail, but a token was issued")
		}
		return nil
	})

	sc.Step(`^the SDK surfaces that error to the caller instead of a generic transport failure$`, func() error {
		if w.lastErr == nil {
			return errors.New("expected an error to surface to the caller")
		}
		// A transport-level failure (DNS, connection refused, TLS) would not
		// mention the controller's own rejection reason; a surfaced API error
		// does.
		if !containsControllerRejection(w.lastErr.Error()) {
			return fmt.Errorf("expected the controller's rejection reason in the error, got: %v", w.lastErr)
		}
		return nil
	})

	sc.Step(`^the client makes (\d+) authenticated requests? to list sites$`, func(ctx context.Context, count int) error {
		httpClient := w.insecureHTTPClient()
		client, err := sdk.NewWithClientCredentials(w.env.BaseURL, w.clientID, w.clientSecret, w.env.OmadacID, httpClient)
		if err != nil {
			return err
		}
		w.sdkClient = client
		page, pageSize := int32(1), int32(1)
		for i := 0; i < count; i++ {
			_, err := client.Openapi().V1().ByOmadacId(w.env.OmadacID).Sites().Get(ctx, &openapi.V1ItemSitesRequestBuilderGetRequestConfiguration{
				QueryParameters: &openapi.V1ItemSitesRequestBuilderGetQueryParameters{Page: &page, PageSize: &pageSize},
			})
			if err != nil {
				return fmt.Errorf("request %d failed: %w", i+1, err)
			}
		}
		return nil
	})

	sc.Step(`^the client makes an authenticated request to list sites$`, func(ctx context.Context) error {
		if w.capture == nil {
			w.capture = &capturingTransport{}
		}
		page, pageSize := int32(1), int32(10)
		_, err := w.sdkClient.Openapi().V1().ByOmadacId(w.env.OmadacID).Sites().Get(ctx, &openapi.V1ItemSitesRequestBuilderGetRequestConfiguration{
			QueryParameters: &openapi.V1ItemSitesRequestBuilderGetQueryParameters{Page: &page, PageSize: &pageSize},
		})
		w.lastErr = err
		return nil
	})

	sc.Step(`^the controller's token endpoint was called exactly (\d+) times?$`, func(count int) error {
		got := w.capture.CountRequestsToPath("/authorize/token")
		if got != count {
			return fmt.Errorf("expected the token endpoint to be called %d time(s), got %d", count, got)
		}
		return nil
	})

	sc.Step(`^the SDK reports an empty access token error without contacting the controller$`, func() error {
		if !errors.Is(w.lastErr, auth.ErrEmptyToken) {
			return fmt.Errorf("expected auth.ErrEmptyToken, got: %v", w.lastErr)
		}
		if got := w.capture.RequestCount(); got != 0 {
			return fmt.Errorf("expected no requests to reach the controller, but observed %d", got)
		}
		return nil
	})

	sc.Step(`^(\d+) goroutines concurrently request an access token from the same provider$`, func(ctx context.Context, n int) error {
		httpClient := w.insecureHTTPClient()
		provider := auth.NewClientCredentialsTokenProvider(w.env.BaseURL, w.clientID, w.clientSecret, w.env.OmadacID, httpClient)

		tokens := make([]string, n)
		errs := make([]error, n)
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(i int) {
				defer wg.Done()
				tokens[i], errs[i] = provider.GetAuthorizationToken(ctx, nil, nil)
			}(i)
		}
		wg.Wait()

		for i, err := range errs {
			if err != nil {
				return fmt.Errorf("goroutine %d: %w", i, err)
			}
		}
		w.concurrentTokens = tokens
		return nil
	})

	sc.Step(`^every goroutine received the same access token$`, func() error {
		if len(w.concurrentTokens) == 0 {
			return errors.New("no tokens recorded")
		}
		first := w.concurrentTokens[0]
		if first == "" {
			return errors.New("expected a non-empty access token")
		}
		for i, tok := range w.concurrentTokens {
			if tok != first {
				return fmt.Errorf("goroutine %d received a different token (%q) than goroutine 0 (%q)", i, tok, first)
			}
		}
		return nil
	})

	// --- tls.feature --------------------------------------------------------

	sc.Step(`^a client configured with the default, certificate-verifying HTTP client$`, func() error {
		w.httpClient = secureVerifyingHTTPClient()
		return nil
	})

	sc.Step(`^a client configured with the SDK's insecure HTTP client$`, func() error {
		w.httpClient = w.insecureHTTPClient()
		return nil
	})

	sc.Step(`^the client calls the controller over HTTPS$`, func(ctx context.Context) error {
		_, err := discoverOmadacID(ctx, w.httpClient, w.env.BaseURL)
		w.lastErr = err
		return nil
	})

	sc.Step(`^the request fails with a certificate verification error$`, func() error {
		if w.lastErr == nil {
			return errors.New("expected a certificate verification error, but the request succeeded")
		}
		var unknownAuth x509.UnknownAuthorityError
		var hostErr x509.HostnameError
		var certInvalid x509.CertificateInvalidError
		if !errors.As(w.lastErr, &unknownAuth) && !errors.As(w.lastErr, &hostErr) && !errors.As(w.lastErr, &certInvalid) {
			return fmt.Errorf("expected a certificate verification error, got: %w", w.lastErr)
		}
		return nil
	})

	sc.Step(`^the request reaches the controller without a certificate verification error$`, func() error {
		if w.lastErr != nil {
			return fmt.Errorf("expected no error, got: %w", w.lastErr)
		}
		return nil
	})

	// --- sites.feature --------------------------------------------------------

	sc.Step(`^an authenticated client for the controller$`, func(ctx context.Context) error {
		httpClient := w.insecureHTTPClient()
		omadacID, err := discoverOmadacID(ctx, httpClient, w.env.BaseURL)
		if err != nil {
			return fmt.Errorf("discovering omadacId: %w", err)
		}
		w.env.OmadacID = omadacID

		client, err := sdk.NewWithClientCredentials(w.env.BaseURL, w.env.ClientID, w.env.ClientSecret, omadacID, httpClient)
		if err != nil {
			return err
		}
		w.sdkClient = client
		return nil
	})

	sc.Step(`^the client requests page (\d+) of sites with a page size of (\d+)$`, func(ctx context.Context, page, pageSize int) error {
		p, ps := int32(page), int32(pageSize)
		resp, err := w.sdkClient.Openapi().V1().ByOmadacId(w.env.OmadacID).Sites().Get(ctx, &openapi.V1ItemSitesRequestBuilderGetRequestConfiguration{
			QueryParameters: &openapi.V1ItemSitesRequestBuilderGetQueryParameters{Page: &p, PageSize: &ps},
		})
		w.lastErr = err
		w.sitesResponse = resp
		if resp != nil {
			w.lastEnvelope = resp
		} else {
			w.lastEnvelope = nil
		}
		return nil
	})

	sc.Step(`^the client requests sites using an unrecognized Omada Cloud Controller ID$`, func(ctx context.Context) error {
		p, ps := int32(1), int32(10)
		resp, err := w.sdkClient.Openapi().V1().ByOmadacId("deliberately-unrecognized-omadac-id").Sites().Get(ctx, &openapi.V1ItemSitesRequestBuilderGetRequestConfiguration{
			QueryParameters: &openapi.V1ItemSitesRequestBuilderGetQueryParameters{Page: &p, PageSize: &ps},
		})
		w.lastErr = err
		w.sitesResponse = resp
		if resp != nil {
			w.lastEnvelope = resp
		} else {
			w.lastEnvelope = nil
		}
		return nil
	})

	sc.Step(`^the response envelope reports success$`, func() error {
		if w.lastErr != nil {
			return fmt.Errorf("request failed: %w", w.lastErr)
		}
		if w.lastEnvelope == nil || w.lastEnvelope.GetErrorCode() == nil {
			return errors.New("response did not contain an errorCode")
		}
		if code := *w.lastEnvelope.GetErrorCode(); code != 0 {
			msg := ""
			if w.lastEnvelope.GetMsg() != nil {
				msg = *w.lastEnvelope.GetMsg()
			}
			return fmt.Errorf("controller reported errorCode %d: %s", code, msg)
		}
		return nil
	})

	sc.Step(`^the response envelope reports a controller-side validation error$`, func() error {
		if w.lastErr != nil {
			return fmt.Errorf("expected the controller's own error envelope, but the SDK returned a transport-level error instead: %w", w.lastErr)
		}
		if w.lastEnvelope == nil || w.lastEnvelope.GetErrorCode() == nil {
			return errors.New("response did not contain an errorCode")
		}
		if code := *w.lastEnvelope.GetErrorCode(); code == 0 {
			return errors.New("expected a non-zero errorCode, got 0 (success)")
		}
		if w.lastEnvelope.GetMsg() == nil || *w.lastEnvelope.GetMsg() == "" {
			return errors.New("expected a non-empty msg describing the validation error")
		}
		return nil
	})

	sc.Step(`^the response contains a page of site summaries$`, func() error {
		if w.sitesResponse == nil || w.sitesResponse.GetResult() == nil {
			return errors.New("response did not contain a result")
		}
		if w.sitesResponse.GetResult().GetData() == nil {
			return errors.New("response result did not contain a data page")
		}
		return nil
	})

	sc.Step(`^the response contains an empty page of site summaries$`, func() error {
		if w.sitesResponse == nil || w.sitesResponse.GetResult() == nil {
			return errors.New("response did not contain a result")
		}
		if got := len(w.sitesResponse.GetResult().GetData()); got != 0 {
			return fmt.Errorf("expected an empty page, got %d site(s)", got)
		}
		return nil
	})

	// --- devices.feature --------------------------------------------------------

	sc.Step(`^the controller's first managed site$`, func(ctx context.Context) error {
		p, ps := int32(1), int32(1)
		resp, err := w.sdkClient.Openapi().V1().ByOmadacId(w.env.OmadacID).Sites().Get(ctx, &openapi.V1ItemSitesRequestBuilderGetRequestConfiguration{
			QueryParameters: &openapi.V1ItemSitesRequestBuilderGetQueryParameters{Page: &p, PageSize: &ps},
		})
		if err != nil {
			return fmt.Errorf("discovering a site: %w", err)
		}
		if resp == nil || resp.GetResult() == nil || len(resp.GetResult().GetData()) == 0 {
			return errors.New("the controller has no managed sites to test against")
		}
		site := resp.GetResult().GetData()[0]
		if site.GetSiteId() == nil {
			return errors.New("site summary did not contain a siteId")
		}
		w.siteID = *site.GetSiteId()
		return nil
	})

	sc.Step(`^the client requests page (\d+) of devices with a page size of (\d+)$`, func(ctx context.Context, page, pageSize int) error {
		p, ps := int32(page), int32(pageSize)
		return w.requestDevices(ctx, &openapi.V1ItemSitesItemDevicesRequestBuilderGetQueryParameters{Page: &p, PageSize: &ps})
	})

	sc.Step(`^the client requests devices with a page size of (\d+) matching search key "([^"]*)"$`, func(ctx context.Context, pageSize int, searchKey string) error {
		p, ps := int32(1), int32(pageSize)
		return w.requestDevices(ctx, &openapi.V1ItemSitesItemDevicesRequestBuilderGetQueryParameters{Page: &p, PageSize: &ps, SearchKey: &searchKey})
	})

	sc.Step(`^the client requests devices with a page size of (\d+) sorted by name ascending$`, func(ctx context.Context, pageSize int) error {
		p, ps := int32(1), int32(pageSize)
		asc := "asc"
		return w.requestDevices(ctx, &openapi.V1ItemSitesItemDevicesRequestBuilderGetQueryParameters{Page: &p, PageSize: &ps, SortsName: &asc})
	})

	sc.Step(`^the client requests devices without specifying a page or page size$`, func(ctx context.Context) error {
		return w.requestDevices(ctx, nil)
	})

	sc.Step(`^the response contains at least (\d+) device$`, func(count int) error {
		if w.devicesResponse == nil || w.devicesResponse.GetResult() == nil {
			return errors.New("response did not contain a result")
		}
		if got := len(w.devicesResponse.GetResult().GetData()); got < count {
			return fmt.Errorf("expected at least %d device(s), got %d", count, got)
		}
		return nil
	})

	sc.Step(`^the response contains an empty page of devices$`, func() error {
		if w.devicesResponse == nil || w.devicesResponse.GetResult() == nil {
			return errors.New("response did not contain a result")
		}
		if got := len(w.devicesResponse.GetResult().GetData()); got != 0 {
			return fmt.Errorf("expected an empty page, got %d device(s)", got)
		}
		return nil
	})

	sc.Step(`^every device in the response has "([^"]*)" in its name$`, func(substr string) error {
		if w.devicesResponse == nil || w.devicesResponse.GetResult() == nil {
			return errors.New("response did not contain a result")
		}
		data := w.devicesResponse.GetResult().GetData()
		if len(data) == 0 {
			return errors.New("expected at least one matching device, got none")
		}
		for _, d := range data {
			name := ""
			if d.GetName() != nil {
				name = *d.GetName()
			}
			if !strings.Contains(strings.ToLower(name), strings.ToLower(substr)) {
				return fmt.Errorf("device %q does not contain search key %q", name, substr)
			}
		}
		return nil
	})

	sc.Step(`^the devices in the response are sorted by name ascending$`, func() error {
		if w.devicesResponse == nil || w.devicesResponse.GetResult() == nil {
			return errors.New("response did not contain a result")
		}
		data := w.devicesResponse.GetResult().GetData()
		names := make([]string, 0, len(data))
		for _, d := range data {
			if d.GetName() != nil {
				names = append(names, *d.GetName())
			}
		}
		if len(names) < 2 {
			return nil // nothing meaningful to assert about ordering
		}
		if !sort.StringsAreSorted(names) {
			return fmt.Errorf("expected device names sorted ascending, got: %v", names)
		}
		return nil
	})

	sc.Step(`^the SDK reports a generic error instead of the controller's specific reason$`, func() error {
		if w.lastErr == nil {
			return errors.New("expected an error (this documents a known SDK limitation), but the request succeeded")
		}
		if containsControllerRejection(w.lastErr.Error()) {
			return fmt.Errorf("expected an opaque, controller-message-free error (documenting the current limitation), but got one that includes controller detail: %v", w.lastErr)
		}
		return nil
	})
}

// requestDevices issues a Devices().Get() call for w.siteID and records the result.
func (w *omadaWorld) requestDevices(ctx context.Context, qp *openapi.V1ItemSitesItemDevicesRequestBuilderGetQueryParameters) error {
	var cfg *openapi.V1ItemSitesItemDevicesRequestBuilderGetRequestConfiguration
	if qp != nil {
		cfg = &openapi.V1ItemSitesItemDevicesRequestBuilderGetRequestConfiguration{QueryParameters: qp}
	}
	resp, err := w.sdkClient.Openapi().V1().ByOmadacId(w.env.OmadacID).Sites().BySiteId(w.siteID).Devices().Get(ctx, cfg)
	w.lastErr = err
	w.devicesResponse = resp
	if resp != nil {
		w.lastEnvelope = resp
	} else {
		w.lastEnvelope = nil
	}
	return nil
}

func containsControllerRejection(msg string) bool {
	// Errors surfaced by auth.ClientCredentialsTokenProvider wrap the
	// controller's own errorCode/msg envelope; a bare transport error would not.
	return strings.Contains(msg, "omada auth:") || strings.Contains(msg, "errorCode")
}
