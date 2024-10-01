package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

/*
 * version is the tagged version of this repository. It is overriden at build time by ldflags.
 * please see the .goreleaser.yml file for more information.
 */
var version = "dev"

// APIError is an error type that exposes additional information about why an API request failed.
type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int
	RawMessage []byte
	retryAfter int
}

// Error provides a user friendly error message.
func (e *APIError) Error() string {
	return fmt.Sprintf("%s - %s", e.Code, e.Message)
}

type clientRequest struct {
	ctx              context.Context
	method           string
	url              string
	body             string
	errorOnNoContent bool
}

func (cr *clientRequest) toHTTPRequest() (*http.Request, error) {
	r, err := http.NewRequestWithContext(
		cr.ctx,
		cr.method,
		cr.url,
		strings.NewReader(cr.body),
	)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", fmt.Sprintf("terraform-provider-vercel/%s", version))
	if cr.body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	return r, nil
}

// doRequest is a helper function for consistently requesting data from Vercel.
// This manages:
// - Setting the default Content-Type for requests with a body
// - Setting the User-Agent
// - Authorization via the Bearer token
// - Converting error responses into an inspectable type
// - Unmarshaling responses
// - Parsing a Retry-After header in the case of rate limits being hit
// - In the case of a rate-limit being hit, trying again after a period of time
func (c *Client) doRequest(req clientRequest, v interface{}) error {
	doRequest := func() error {
		r, err := req.toHTTPRequest()
		if err != nil {
			return err
		}
		return c._doRequest(r, v, req.errorOnNoContent)
	}
	err := doRequest()
	for retries := 0; retries < 3; retries++ {
		var apiErr *APIError
		if errors.As(err, &apiErr) && // we received an api error
			apiErr.StatusCode == http.StatusTooManyRequests && // and it was a rate limit
			apiErr.retryAfter > 0 && // and there was a retry time
			apiErr.retryAfter < 5*60 { // and the retry time is less than 5 minutes
			tflog.Error(req.ctx, "Rate limit was hit", map[string]interface{}{
				"error":      apiErr,
				"retryAfter": apiErr.retryAfter,
			})
			timer := time.NewTimer(time.Duration(apiErr.retryAfter) * time.Second)
			select {
			case <-req.ctx.Done():
				timer.Stop()
				return req.ctx.Err()
			case <-timer.C:
			}
			r, err := req.toHTTPRequest()
			if err != nil {
				return err
			}
			if err = c._doRequest(r, v, req.errorOnNoContent); err == nil {
				return nil
			}
		} else {
			break
		}
	}

	return err
}

func (c *Client) _doRequest(req *http.Request, v interface{}, errorOnNoContent bool) error {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	resp, err := c.http().Do(req)
	if err != nil {
		return fmt.Errorf("error doing http request: %w", err)
	}

	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode >= 300 {
		if string(responseBody) == "" {
			return &APIError{
				StatusCode: resp.StatusCode,
			}
		}
		var errorResponse APIError
		err = json.Unmarshal(responseBody, &struct {
			Error *APIError `json:"error"`
		}{
			Error: &errorResponse,
		})
		if err != nil {
			return fmt.Errorf("error unmarshaling response for status code %d: %w", resp.StatusCode, err)
		}
		errorResponse.StatusCode = resp.StatusCode
		errorResponse.RawMessage = responseBody
		errorResponse.retryAfter = 1000 // set a sensible default for retrying. This is in milliseconds.
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfterRaw := resp.Header.Get("Retry-After")
			if retryAfterRaw != "" {
				retryAfter, err := strconv.Atoi(retryAfterRaw)
				if err == nil && retryAfter > 0 {
					errorResponse.retryAfter = retryAfter
				}
			}
		}
		return &errorResponse
	}

	if v == nil {
		return nil
	}

	if errorOnNoContent && resp.StatusCode == http.StatusNoContent {
		return &APIError{
			StatusCode: resp.StatusCode,
			Code:       "no_content",
			Message:    "No content",
		}
	}

	err = json.Unmarshal(responseBody, v)
	if err != nil {
		return fmt.Errorf("error unmarshaling response %s: %w", responseBody, err)
	}

	return nil
}
