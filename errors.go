package wajub

import "fmt"

// WajubError is the base error for API failures.
type WajubError struct {
	Message    string
	Code       string
	HTTPStatus int
	Errors     map[string]string
	Raw        map[string]any
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

	var fieldErrors map[string]string
	if rawErrors, ok := body["errors"].(map[string]any); ok {
		fieldErrors = make(map[string]string, len(rawErrors))
		for field, value := range rawErrors {
			switch v := value.(type) {
			case []any:
				if len(v) > 0 {
					fieldErrors[field] = fmt.Sprint(v[0])
				}
			default:
				fieldErrors[field] = fmt.Sprint(v)
			}
		}
	}

	base := WajubError{
		Message:    message,
		Code:       code,
		HTTPStatus: status,
		Errors:     fieldErrors,
		Raw:        body,
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
