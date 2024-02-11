package restplay

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestGetClientID(t *testing.T) {
	const baseURL = "http://example.com"

	tests := map[string]struct {
		Method           string
		ContentType      string
		ClientID         string
		ExpectedErrorSub string
		UseBasicAuth     bool
		UseBearerToken   bool
	}{
		"should find client_id for form POSTed requests without error": {
			Method:      http.MethodPost,
			ContentType: formContentType,
			ClientID:    "robbie-client-id",
		},
		"should find client_id in the URL of non-body requests without error": {
			ClientID: "robbie-other-client-id",
		},
		"should find client_id in BasicAuth of requests without error": {
			ClientID:     "robbie-BasicAuth-client-id",
			UseBasicAuth: true,
		},
		"should find client_id in Bearer token of requests without error": {
			ClientID:       "robbie-BearerToken-client-id",
			UseBearerToken: true,
		},
		"should return error on GET without any client_id provided": {
			ExpectedErrorSub: "failed to find client_id",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Test Case Setup
			var (
				form         url.Values
				req          *http.Request
				err          error
				bodyAsString string
				method       = func() string {
					if tt.Method != "" {
						return tt.Method
					}
					return http.MethodGet
				}()
			)
			switch {
			case tt.UseBearerToken:
				req, err = http.NewRequest(method, baseURL, nil)
				if err == nil {
					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s.othertokenstuffhere", tt.ClientID))
				}
			case tt.UseBasicAuth:
				req, err = http.NewRequest(method, baseURL, nil)
				if err == nil {
					req.SetBasicAuth(tt.ClientID, "password")
				}
			default:
				form = make(url.Values, 1)
				form.Set(clientIDKey, tt.ClientID)
				switch tt.Method {
				case http.MethodPost, http.MethodPut, http.MethodPatch:
					bodyAsString = form.Encode()
					req, err = http.NewRequest(method, baseURL, strings.NewReader(bodyAsString))
					if err == nil {
						req.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyAsString)))
					}
				default:
					u := fmt.Sprintf("%s?%s", baseURL, form.Encode())
					req, err = http.NewRequest(method, u, nil)
				}
			}
			if err != nil {
				t.Fatalf("failed to create request for test: %s", err)
			}

			if tt.ContentType != "" {
				req.Header.Set(contentTypeHeaderKey, tt.ContentType)
			}

			// Now do the actual thing: GetClientID
			var actualClientID string
			actualClientID, _, err = GetClientID(req)

			// Assert all of our expectations
			if len(tt.ExpectedErrorSub) > 0 {
				// error expected
				if err != nil {
					errStr := err.Error()
					if !strings.Contains(errStr, tt.ExpectedErrorSub) {
						t.Errorf("Expected error:\n  %s\n\nTo contain:\n  %q", errStr, tt.ExpectedErrorSub)
					}
				} else {
					t.Errorf("Expected an error that contained: %q but error was nil", tt.ExpectedErrorSub)
				}
			} else {
				// no error expected
				if err != nil {
					t.Errorf("No error expected but got: %q", err)
				}
			}
			if actualClientID != tt.ClientID {
				t.Errorf("GetClientID() got = %q, want %q", actualClientID, tt.ClientID)
			}
			if len(bodyAsString) > 0 {
				// we now must test that we can read our request body again
				var afterBody []byte
				if afterBody, err = io.ReadAll(req.Body); err != nil {
					t.Errorf("Unable to read request body after passed to GetClientID(): %s", err)
				}
				if bodyAsString != string(afterBody) {
					t.Errorf("Request body after passed to GetClientID changed:\n  Original: %q\n  After:   %q", bodyAsString, afterBody)
				}
			}
		})
	}
}
