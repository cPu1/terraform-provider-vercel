package client

import (
	"errors"
	"net/http"
)

// NotFound detects if an error returned by the Vercel API was the result of an entity not existing.
func NotFound(err error) bool {
	return hasStatus(err, http.StatusNotFound)
}

func noContent(err error) bool {
	return hasStatus(err, http.StatusNoContent)
}

func hasStatus(err error, statusCode int) bool {
	var apiErr *APIError
	return err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == statusCode
}
