package license

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// roundTripFunc .
type roundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// newTestClient returns *http.Client with Transport replaced to avoid making real calls.
func newTestClient(fn roundTripFunc) *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: fn,
		},
	}
}

func TestCheckQuota(t *testing.T) {
	testCases := map[string]struct {
		license               string
		serverResponse        string
		serverResponseCode    int
		serverResponseContent string
		wantQuota             int
		wantError             bool
	}{
		"success": {
			license:               "***REMOVED***",
			serverResponse:        "{\"quota\":256}",
			serverResponseCode:    http.StatusOK,
			serverResponseContent: "application/json",
			wantQuota:             256,
		},
		"404": {
			serverResponseCode: http.StatusNotFound,
			wantError:          true,
		},
		"HTML not JSON": {
			serverResponseCode:    http.StatusOK,
			serverResponseContent: "text/html",
			wantError:             true,
		},
		"promise JSON but actually HTML": {
			serverResponseCode:    http.StatusOK,
			serverResponse:        "<html><head></head></html>",
			serverResponseContent: "application/json",
			wantError:             true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			client := newTestClient(func(req *http.Request) *http.Response {
				r := &http.Response{
					StatusCode: tc.serverResponseCode,
					Body:       io.NopCloser(bytes.NewBufferString(tc.serverResponse)),
					Header:     make(http.Header),
				}
				r.Header.Set("Content-Type", tc.serverResponseContent)
				return r
			})

			resp, err := client.CheckQuota(context.Background(), CheckQuotaRequest{
				Action:  test,
				License: tc.license,
			})

			if tc.wantError {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tc.wantQuota, resp.Quota)
		})
	}
}

func Test_licenseURL(t *testing.T) {
	assert.Equal(t, "https://license.confidential.cloud/api/v1/license", licenseURL().String())
}