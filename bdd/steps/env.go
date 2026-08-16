package steps

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// testEnv holds the live-controller configuration used by the BDD suite, loaded
// from the repo-root .env file (falling back to real process environment
// variables of the same name).
type testEnv struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	TLSVerify    bool
	OmadacID     string
}

func loadTestEnv() (*testEnv, error) {
	values := map[string]string{}
	if f, err := os.Open(filepath.Join("..", "..", ".env")); err == nil {
		defer func() { _ = f.Close() }()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), `"`)
			values[key] = val
		}
	}
	get := func(key string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return values[key]
	}

	env := &testEnv{
		BaseURL:      strings.TrimRight(get("BASE_URL"), "/"),
		ClientID:     get("CLIENT_ID"),
		ClientSecret: get("CLIENT_SECRET"),
		TLSVerify:    strings.EqualFold(get("TLS_VERIFY"), "true"),
	}
	if env.BaseURL == "" || env.ClientID == "" || env.ClientSecret == "" {
		return nil, fmt.Errorf("BASE_URL, CLIENT_ID and CLIENT_SECRET must be set (via .env at the repo root or the environment)")
	}
	return env, nil
}

// controllerInfo mirrors the unauthenticated GET /api/info bootstrap endpoint, which
// is how a caller discovers a controller's Omada Cloud Controller ID before it can
// make any authenticated Open API call.
type controllerInfo struct {
	ErrorCode int32 `json:"errorCode"`
	Result    struct {
		OmadacID string `json:"omadacId"`
	} `json:"result"`
}

func discoverOmadacID(ctx context.Context, httpClient *http.Client, baseURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/info", nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var info controllerInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("decoding /api/info response: %w", err)
	}
	if info.Result.OmadacID == "" {
		return "", fmt.Errorf("/api/info response did not contain an omadacId (errorCode %d)", info.ErrorCode)
	}
	return info.Result.OmadacID, nil
}
