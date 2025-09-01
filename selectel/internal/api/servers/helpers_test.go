package servers

import (
	"io"
	"net/http"
	"strings"
)

// roundTripFunc lets us use a function as an http.RoundTripper.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// newFakeResponse creates a fake *http.Response with the provided status and body.
func newFakeResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// newFakeTransport returns a fake transport with the given response and error.
func newFakeTransport(resp *http.Response, err error) roundTripFunc {
	return roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return resp, err
	})
}

// newFakeClient creates a new ServiceClient with the given endpoint and transport.
func newFakeClient(endpoint string, transport http.RoundTripper) *ServiceClient {
	return &ServiceClient{
		HTTPClient: &http.Client{Transport: transport},
		Endpoint:   endpoint,
	}
}

const (
	invalidJSONBody = `{
			"result": [
				invalid
			]
		}`

	httpErrorBody    = "Not Found"
	httpErrorMessage = "got the 404 status code from the server: Not Found"
)
