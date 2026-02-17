package auth

import "net/http"

// defaultTransport is used by HTTP clients created in this package.
//
// It is a var (not a const) so tests can replace it to stub out network calls.
var defaultTransport http.RoundTripper = http.DefaultTransport
