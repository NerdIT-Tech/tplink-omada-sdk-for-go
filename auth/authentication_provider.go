package auth

import (
	"context"
	"errors"
	"fmt"

	abs "github.com/microsoft/kiota-abstractions-go"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
)

// authorizationHeader is the header Omada expects credentials on. Despite the token
// response reporting "tokenType": "bearer", the controller does not accept the
// standard "Authorization: Bearer <token>" scheme — it requires
// "Authorization: AccessToken=<token>" instead. Sending the standard Bearer scheme
// is accepted by the transport but silently rejected by the controller with an
// unrelated-looking "access token has expired" error.
const authorizationHeader = "Authorization"

// ErrEmptyToken is returned by AuthenticateRequest when the configured
// AccessTokenProvider yields an empty token. Sending the request anyway would reach
// the controller with no Authorization header at all; the controller then rejects it
// with an HTTP 401 whose own error envelope is discarded by the generated request
// adapter (no error mapping is registered for that status), surfacing only an opaque
// "unexpected status code" error far from the real cause. Failing locally, before any
// request is sent, gives the caller an immediate, actionable reason instead.
var ErrEmptyToken = errors.New("omada sdk: access token provider returned an empty token")

// AuthenticationProvider authenticates requests using an Omada access token,
// obtained from the given AccessTokenProvider (StaticTokenProvider or
// ClientCredentialsTokenProvider), via Omada's non-standard
// "Authorization: AccessToken=<token>" header scheme.
type AuthenticationProvider struct {
	tokenProvider absauth.AccessTokenProvider
}

var _ absauth.AuthenticationProvider = (*AuthenticationProvider)(nil)

// NewAuthenticationProvider creates an AuthenticationProvider backed by
// tokenProvider.
func NewAuthenticationProvider(tokenProvider absauth.AccessTokenProvider) *AuthenticationProvider {
	return &AuthenticationProvider{tokenProvider: tokenProvider}
}

// AuthenticateRequest attaches an Omada access token to the outgoing request.
func (p *AuthenticationProvider) AuthenticateRequest(ctx context.Context, request *abs.RequestInformation, additionalAuthenticationContext map[string]interface{}) error {
	if request == nil {
		return errors.New("request is nil")
	}
	if request.Headers == nil {
		request.Headers = abs.NewRequestHeaders()
	}
	if p.tokenProvider == nil {
		return errors.New("this provider needs to be initialized with an access token provider")
	}
	if request.Headers.ContainsKey(authorizationHeader) {
		return nil
	}

	uri, err := request.GetUri()
	if err != nil {
		return err
	}
	token, err := p.tokenProvider.GetAuthorizationToken(ctx, uri, additionalAuthenticationContext)
	if err != nil {
		return err
	}
	if token == "" {
		return ErrEmptyToken
	}
	request.Headers.Add(authorizationHeader, fmt.Sprintf("AccessToken=%s", token))
	return nil
}
