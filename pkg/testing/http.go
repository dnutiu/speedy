package testing

import "net/http"

// RoundTripFunc is a mock for testing httpClient
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip is a mock for testing httpClient
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

//NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}
