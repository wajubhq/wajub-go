package wajub

import "context"

type ListenService struct{ base baseClient }

// Config always returns an error: the API has retired it.
//
// Deprecated: The API answers 410 Gone. Forward webhooks to your machine
// with the Wajub CLI's `wajub listen`.
func (s *ListenService) Config(ctx context.Context) (map[string]any, error) {
	res, err := s.base.get(ctx, "/listen/config", nil)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "config", "listen"), nil
}

// Auth always returns an error: the API has retired it.
//
// Deprecated: The API answers 410 Gone. Forward webhooks to your machine
// with the Wajub CLI's `wajub listen`.
func (s *ListenService) Auth(ctx context.Context, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/listen/auth", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "auth", "listen"), nil
}
