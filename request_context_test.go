package restplay

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// test case types
type argsGetClientID struct {
	Method           string
	ContentType      string
	ClientID         string
	ExpectedErrorSub string
	UseBasicAuth     bool
	UseBearerToken   bool
}

func TestGetClientID(t *testing.T) {
	const baseURL = "https://example.com"

	tests := map[string]argsGetClientID{
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
			ClientID:       "robbie-BearerToken-GET-client-id",
			UseBearerToken: true,
		},
		"should find client_id in Bearer token of PUT requests without error": {
			Method:         http.MethodPut,
			ClientID:       "robbie-BearerToken-PUT-client-id",
			UseBearerToken: true,
		},
		"should return error on GET without any client_id provided": {
			ExpectedErrorSub: "failed to find client_id",
		},
		"should return error on PATCH without any client_id provided": {
			Method:           http.MethodPatch,
			ContentType:      formContentType,
			ExpectedErrorSub: "failed to find client_id",
		},
		"should return error on BasicAuth token requests without any client_id provided": {
			UseBasicAuth:     true,
			ExpectedErrorSub: "failed to find client_id",
		},
		"should return error on Bearer token requests without any client_id provided": {
			UseBearerToken:   true,
			ExpectedErrorSub: "invalid token",
		},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			// Test Case Setup
			req, err, bodyAsString := setupGetClientID(args, baseURL)
			if err != nil {
				t.Fatalf("failed to create request for test: %s", err)
			}

			// Now do the actual thing: GetClientID
			var actualClientID string
			actualClientID, req, err = GetClientID(req)

			// Assert all of our expectations
			if len(args.ExpectedErrorSub) > 0 {
				// error expected
				if err != nil {
					errStr := err.Error()
					if !strings.Contains(errStr, args.ExpectedErrorSub) {
						t.Errorf("\nExpected error:\n  %q\n\nTo contain:\n  %q", errStr, args.ExpectedErrorSub)
					}
				} else {
					t.Errorf("Expected an error that contained: %q but error was nil", args.ExpectedErrorSub)
				}
			} else {
				// no error expected
				if err != nil {
					t.Errorf("No error expected but got: %q", err)
				}
			}
			if actualClientID != args.ClientID {
				t.Errorf("GetClientID() got = %q, want %q", actualClientID, args.ClientID)
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

func setupGetClientID(args argsGetClientID, baseURL string) (*http.Request, error, string) {
	var (
		form         url.Values
		req          *http.Request
		err          error
		bodyAsString string
		method       = func() string {
			if args.Method != "" {
				return args.Method
			}
			return http.MethodGet
		}()
	)
	switch {
	case args.UseBearerToken:
		req, err = http.NewRequest(method, baseURL, nil)
		if err == nil {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s.othertokenstuffhere", args.ClientID))
		}
	case args.UseBasicAuth:
		req, err = http.NewRequest(method, baseURL, nil)
		if err == nil {
			req.SetBasicAuth(args.ClientID, "password")
		}
	default:
		form = make(url.Values, 1)
		form.Set(clientIDKey, args.ClientID)
		switch args.Method {
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
	if err == nil && args.ContentType != "" {
		req.Header.Set(contentTypeHeaderKey, args.ContentType)
	}
	return req, err, bodyAsString
}
