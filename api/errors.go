package api

import (
	"errors"
	"net/http"
)

// HTTPStatus returns the HTTP status of an API error, or 0 if err isn't an
// HTTP error from the server (e.g. a network failure).
func HTTPStatus(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Type == ErrorTypeHTTPError {
		return apiErr.Status
	}
	return 0
}

// IsForbidden reports whether the server refused the request with 403: the
// session is valid but the user's role may not do this.
func IsForbidden(err error) bool {
	return HTTPStatus(err) == http.StatusForbidden
}
