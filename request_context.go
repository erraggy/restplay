package restplay

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

const (
	bearerPrefix         = "Bearer "
	formContentType      = "application/x-www-form-urlencoded"
	clientIDKey          = "client_id"
	contentTypeHeaderKey = "Content-Type"
)

var (
	// ErrInvalidBearerToken is returned is a Bearer token is defined but invalid
	ErrInvalidBearerToken = errors.New("restplay: invalid token")
	// ErrNilRequest is returned if a nil *http.Request is received
	ErrNilRequest = errors.New("restplay: cannot get client_id from nil request")
	// ErrMissingClientID is the default error returned if no client_id is found
	ErrMissingClientID = errors.New("restplay: failed to find client_id in request")
)

// GetClientID will attempt to extract the client_id from the request.
// It returns the client_id, and possible error.
func GetClientID(req *http.Request) (string, error) {
	if req == nil {
		return "", ErrNilRequest
	}

	// first attempt basic-auth
	if clientID, _, ok := req.BasicAuth(); ok && clientID != "" {
		return clientID, nil
	}

	// next check bearer token
	if auth := req.Header.Get("Authorization"); strings.HasPrefix(auth, bearerPrefix) {
		return GetClientIDFromBearerToken(strings.TrimPrefix(auth, bearerPrefix))
	}

	// finally check in the request form
	// before accessing the form we may need to read the body so
	switch req.Method {
	case http.MethodPost, http.MethodPatch, http.MethodPut:
		// if the content-type is application/x-www-form-urlencoded then we look in the PostForm
		mimetype, _, _ := mime.ParseMediaType(req.Header.Get(contentTypeHeaderKey))
		if mimetype == formContentType && req.Body != nil {
			if req.Form == nil {
				// here is the only case where we will need to copy the request body
				bodyBytes, err := io.ReadAll(req.Body)
				if err != nil {
					// this fails to reset the body, but not my fault
					return "", fmt.Errorf("restplay: failed to read request body: %w", err)
				}
				// since we had to read the body in order to copy its content,
				// we must reset it before the following call to ParseForm()
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				if err = req.ParseForm(); err != nil {
					// reset body before returning the error, since the ParseForm() may
					// have read the body again
					req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
					return "", fmt.Errorf("restplay: failed to parse request form from body: %w", err)
				}
				// we successfully parsed the form, so we can now reset the body
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		} else {
			// no need to touch the request body, so this will protect from nil access
			if req.Form == nil {
				req.Form = make(url.Values)
			}
		}
	default:
		if req.Form == nil {
			// this call to ParseFrom() will not touch the body because the request's method doesn't call for it
			if err := req.ParseForm(); err != nil {
				return "", fmt.Errorf("restplay: failed to parse request form from URL: %w", err)
			}
		}
	}

	// it is now safe to access the request's form
	if clientID := req.Form.Get(clientIDKey); clientID != "" {
		return clientID, nil
	}

	// all known cases exhausted without finding a client_id
	return "", ErrMissingClientID
}

// GetClientIDFromBearerToken will attempt to parse/validate the token and return the identity
func GetClientIDFromBearerToken(token string) (string, error) {
	fields := strings.Split(token, ".")
	if len(fields) != 2 {
		return "", ErrInvalidBearerToken
	}
	clientID := fields[0]
	if clientID == "" {
		return "", ErrInvalidBearerToken
	}
	return clientID, nil
}
