package wajub

import (
	"errors"
	"fmt"
)

// ErrEmptyID is returned by a method given an empty id, before anything is
// sent. Check it with errors.Is.
var ErrEmptyID = errors.New("wajub: id is empty")

// WajubError is the base error for API failures.
type WajubError struct {
	Message    string
	Code       string
	HTTPStatus int
	// Errors holds the first validation message of each field.
	Errors map[string]string
	// FieldErrors holds every validation message of each field, in the
	// order the API sent them.
	FieldErrors map[string][]string
	Raw         map[string]any
}

func (e *WajubError) Error() string {
	return e.Message
}

// AuthenticationError is returned for HTTP 401 responses.
type AuthenticationError struct{ WajubError }

// PermissionError is returned for HTTP 403 responses.
type PermissionError struct{ WajubError }

// NotFoundError is returned for HTTP 404 responses.
type NotFoundError struct{ WajubError }

// InvalidRequestError is returned for HTTP 400/422 responses.
type InvalidRequestError struct{ WajubError }

// RateLimitError is returned for HTTP 429 responses.
type RateLimitError struct {
	WajubError
	RetryAfter int
}

// APIConnectionError is returned when the HTTP transport fails.
type APIConnectionError struct {
	WajubError
	// Err is the underlying transport error, if any. Use errors.Is/errors.As
	// (e.g. errors.Is(err, context.DeadlineExceeded)) to inspect it.
	Err error
}

// Unwrap exposes the underlying transport error for errors.Is/errors.As.
func (e *APIConnectionError) Unwrap() error { return e.Err }

// WebhookSignatureVerificationError is returned when webhook verification fails.
type WebhookSignatureVerificationError struct {
	Message string
}

func (e *WebhookSignatureVerificationError) Error() string {
	return e.Message
}

// AsAPIError returns the shared *WajubError underlying any typed Wajub API
// error (AuthenticationError, PermissionError, NotFoundError, InvalidRequestError,
// RateLimitError, APIConnectionError, or a bare *WajubError), allowing callers to
// handle all API error types uniformly without a type switch of their own.
func AsAPIError(err error) (*WajubError, bool) {
	switch e := err.(type) {
	case *WajubError:
		return e, true
	case *AuthenticationError:
		return &e.WajubError, true
	case *PermissionError:
		return &e.WajubError, true
	case *NotFoundError:
		return &e.WajubError, true
	case *InvalidRequestError:
		return &e.WajubError, true
	case *RateLimitError:
		return &e.WajubError, true
	case *APIConnectionError:
		return &e.WajubError, true
	default:
		return nil, false
	}
}

func errorFromResponse(status int, body map[string]any, retryAfter int) error {
	message := fmt.Sprintf("Request failed (%d)", status)
	if m, ok := body["message"].(string); ok {
		message = m
	}

	code := fmt.Sprintf("http_%d", status)
	if c, ok := body["error_code"].(string); ok {
		code = c
	} else if c, ok := body["code"].(string); ok {
		code = c
	}

	var firstErrors map[string]string
	var fieldErrors map[string][]string
	if rawErrors, ok := body["errors"].(map[string]any); ok {
		firstErrors = make(map[string]string, len(rawErrors))
		fieldErrors = make(map[string][]string, len(rawErrors))
		for field, value := range rawErrors {
			switch v := value.(type) {
			case []any:
				if len(v) > 0 {
					messages := make([]string, len(v))
					for i, message := range v {
						messages[i] = fmt.Sprint(message)
					}
					firstErrors[field] = messages[0]
					fieldErrors[field] = messages
				}
			default:
				firstErrors[field] = fmt.Sprint(v)
				fieldErrors[field] = []string{fmt.Sprint(v)}
			}
		}
	}

	base := WajubError{
		Message:     message,
		Code:        code,
		HTTPStatus:  status,
		Errors:      firstErrors,
		FieldErrors: fieldErrors,
		Raw:         body,
	}

	switch status {
	case 401:
		return &AuthenticationError{base}
	case 403:
		return &PermissionError{base}
	case 404:
		return &NotFoundError{base}
	case 429:
		return &RateLimitError{WajubError: base, RetryAfter: retryAfter}
	case 400, 422:
		return &InvalidRequestError{base}
	default:
		return &base
	}
}
