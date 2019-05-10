package provider

import (
	"fmt"
	"log"
	"net/http"

	"github.com/buildkite/go-buildkite/buildkite"
)

func newBuildkiteClient(apiToken string) (*buildkite.Client, error) {
	config, err := buildkite.NewTokenConfig(apiToken, false)
	if err != nil {
		return nil, err
	}
	httpClient := config.Client()

	// Set a User-Agent header on every request, so Buildkite knows who is calling.
	userAgent := fmt.Sprintf("terraform-provider-buildkite/%s (commit %s)", version(), gitCommit)
	httpClient.Transport = &userAgentRoundTripper{
		userAgent: userAgent,
		inner:     httpClient.Transport,
	}

	return buildkite.NewClient(httpClient), nil
}

type userAgentRoundTripper struct {
	inner     http.RoundTripper
	userAgent string
}

func (rt *userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	log.Printf("[TRACE] HTTP %s request to %s", req.Method, req.URL.String())
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", rt.userAgent)
	}
	return rt.inner.RoundTrip(req)
}
